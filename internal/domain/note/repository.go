package note

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Repository manages notes on the filesystem under a single root directory.
//
// Notes are stored as plain markdown files. There is no database, no index,
// no metadata layer — just files in directories. This keeps notes greppable,
// editable by hand, and trivially syncable via Syncthing/iCloud/etc.
type Repository struct {
	root string
}

// NewRepository returns a Repository rooted at the given directory.
// The directory is created lazily when notes are written.
func NewRepository(root string) *Repository {
	return &Repository{root: root}
}

// Root returns the absolute root directory for notes.
func (r *Repository) Root() string {
	return r.root
}

// EntityDir returns the directory that holds notes for the given entity.
// It does not create the directory.
func (r *Repository) EntityDir(et EntityType, entityUUID string) string {
	return filepath.Join(r.root, string(et), entityUUID)
}

// List returns all notes for the given entity, newest-first by filename.
// Returns an empty slice (not an error) if the entity has no notes.
func (r *Repository) List(et EntityType, entityUUID string) ([]Note, error) {
	dir := r.EntityDir(et, entityUUID)
	return r.listDir(dir, et, entityUUID)
}

// ListAll returns every note across every entity, newest-first by filename.
func (r *Repository) ListAll() ([]Note, error) {
	var all []Note

	entityTypes := []EntityType{EntityTask, EntityProject, EntityArea}
	for _, et := range entityTypes {
		typeDir := filepath.Join(r.root, string(et))
		entries, err := os.ReadDir(typeDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			notes, err := r.listDir(filepath.Join(typeDir, e.Name()), et, e.Name())
			if err != nil {
				return nil, err
			}
			all = append(all, notes...)
		}
	}

	sortNotesNewestFirst(all)
	return all, nil
}

func (r *Repository) listDir(dir string, et EntityType, entityUUID string) ([]Note, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var notes []Note
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		n, ok := parseFilename(name)
		if !ok {
			continue
		}
		n.Path = filepath.Join(dir, name)
		n.EntityType = et
		n.EntityUUID = entityUUID
		notes = append(notes, n)
	}

	sortNotesNewestFirst(notes)
	return notes, nil
}

// Create writes a new note for the given entity and returns it.
//
// The filename is `YYYYMMDD--<slug>.md` where slug is derived from title.
// If a note with that name already exists for today, a numeric suffix is
// appended (`-2`, `-3`, ...) until a free name is found.
//
// The body is written verbatim. If body is empty, a small header
// (`# <title>\n\n<YYYY-MM-DD>\n`) is written so the file is not empty.
func (r *Repository) Create(et EntityType, entityUUID, title, body string) (*Note, error) {
	if !ValidEntityType(et) {
		return nil, fmt.Errorf("invalid entity type: %q", et)
	}
	if entityUUID == "" {
		return nil, errors.New("entity uuid is required")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.New("title is required")
	}

	dir := r.EntityDir(et, entityUUID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	now := time.Now()
	datePrefix := now.Format("20060102")
	slug := slugify(title)

	base := fmt.Sprintf("%s--%s", datePrefix, slug)
	filename := base + ".md"
	for i := 2; ; i++ {
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			break
		} else if err != nil {
			return nil, err
		}
		filename = fmt.Sprintf("%s-%d.md", base, i)
	}

	path := filepath.Join(dir, filename)
	content := body
	if content == "" {
		content = fmt.Sprintf("# %s\n\n%s\n", title, now.Format("2006-01-02"))
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil, err
	}

	n, ok := parseFilename(filename)
	if !ok {
		// Should never happen — we just generated the name.
		return nil, fmt.Errorf("internal: failed to parse generated filename %q", filename)
	}
	n.Path = path
	n.EntityType = et
	n.EntityUUID = entityUUID
	return &n, nil
}

// HasNotesByUUIDs checks which entity UUIDs have at least one note file.
func (r *Repository) HasNotesByUUIDs(et EntityType, uuids []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, uuid := range uuids {
		dir := r.EntityDir(et, uuid)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				result[uuid] = true
				break
			}
		}
	}
	return result, nil
}

// Search scans notes for the given (case-insensitive) query substring.
//
// If et is empty, the entire notes tree is searched. Otherwise only notes
// for the given entity are scanned.
func (r *Repository) Search(et EntityType, entityUUID, query string) ([]SearchMatch, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("query is required")
	}

	var notes []Note
	var err error
	if et == "" {
		notes, err = r.ListAll()
	} else {
		notes, err = r.List(et, entityUUID)
	}
	if err != nil {
		return nil, err
	}

	needle := strings.ToLower(query)
	var matches []SearchMatch
	for _, n := range notes {
		f, err := os.Open(n.Path)
		if err != nil {
			return nil, err
		}
		scanner := bufio.NewScanner(f)
		// Allow long lines (default 64KB is fine for markdown but be safe).
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			line := scanner.Text()
			if strings.Contains(strings.ToLower(line), needle) {
				matches = append(matches, SearchMatch{
					Note:    n,
					Line:    lineNo,
					Content: line,
				})
			}
		}
		closeErr := f.Close()
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
	}

	return matches, nil
}

// parseFilename extracts the date and title from a filename of the form
// `YYYYMMDD--<slug>.md` (with optional `-<n>` suffix from collisions).
// Returns ok=false if the filename does not match.
func parseFilename(filename string) (Note, bool) {
	name := strings.TrimSuffix(filename, ".md")
	const sep = "--"
	idx := strings.Index(name, sep)
	if idx != 8 {
		return Note{}, false
	}
	datePart := name[:idx]
	slugPart := name[idx+len(sep):]
	if slugPart == "" {
		return Note{}, false
	}
	t, err := time.ParseInLocation("20060102", datePart, time.Local)
	if err != nil {
		return Note{}, false
	}
	// Strip trailing -<n> collision suffix from the title display.
	title := slugPart
	if i := strings.LastIndex(title, "-"); i >= 0 {
		suffix := title[i+1:]
		allDigits := suffix != ""
		for _, r := range suffix {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			title = title[:i]
		}
	}
	title = strings.ReplaceAll(title, "-", " ")
	return Note{
		Filename: filename,
		Title:    title,
		Date:     t,
	}, true
}

// slugify converts a title into a filesystem-safe slug:
// lowercase, alphanumeric + dashes, no leading/trailing/repeated dashes.
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	lastDash := true
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "note"
	}
	return out
}

// sortNotesNewestFirst sorts notes by filename descending, which puts the
// newest dates first and provides a stable order within the same day.
func sortNotesNewestFirst(notes []Note) {
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].Filename > notes[j].Filename
	})
}
