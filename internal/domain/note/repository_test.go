package note

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Kickoff":            "kickoff",
		"Kickoff Notes":      "kickoff-notes",
		"Hello, World!":      "hello-world",
		"  spaced  out  ":    "spaced-out",
		"über/cool":          "ber-cool",
		"!!!":                "note",
		"":                   "note",
		"already-a-slug":     "already-a-slug",
		"MixedCASE 123":      "mixedcase-123",
		"--leading-dashes--": "leading-dashes",
		"multiple   spaces":  "multiple-spaces",
	}
	for in, want := range cases {
		got := slugify(in)
		if got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseFilename(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		n, ok := parseFilename("20260407--kickoff.md")
		if !ok {
			t.Fatal("expected ok")
		}
		if n.Filename != "20260407--kickoff.md" {
			t.Errorf("filename = %q", n.Filename)
		}
		if n.Title != "kickoff" {
			t.Errorf("title = %q", n.Title)
		}
		if n.Date.Year() != 2026 || n.Date.Month() != 4 || n.Date.Day() != 7 {
			t.Errorf("date = %v", n.Date)
		}
	})

	t.Run("collision suffix stripped", func(t *testing.T) {
		n, ok := parseFilename("20260407--kickoff-2.md")
		if !ok {
			t.Fatal("expected ok")
		}
		if n.Title != "kickoff" {
			t.Errorf("title = %q, want kickoff", n.Title)
		}
	})

	t.Run("multiword slug", func(t *testing.T) {
		n, ok := parseFilename("20260407--my-first-note.md")
		if !ok {
			t.Fatal("expected ok")
		}
		if n.Title != "my first note" {
			t.Errorf("title = %q", n.Title)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		bad := []string{
			"kickoff.md",
			"2026-04-07--kickoff.md",
			"20260407-kickoff.md",
			"20260407--.md",
			"notmd.txt",
			"20261301--bad-month.md",
		}
		for _, name := range bad {
			if _, ok := parseFilename(name); ok {
				t.Errorf("expected parseFilename(%q) to fail", name)
			}
		}
	})
}

func TestRepositoryCreateAndList(t *testing.T) {
	root := t.TempDir()
	repo := NewRepository(root)

	const uuid = "abc-123"

	// Empty list
	notes, err := repo.List(EntityTask, uuid)
	if err != nil {
		t.Fatalf("List on empty: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("empty list got %d notes", len(notes))
	}

	// Create one
	n1, err := repo.Create(EntityTask, uuid, "Kickoff", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !strings.HasSuffix(n1.Filename, "--kickoff.md") {
		t.Errorf("filename = %q", n1.Filename)
	}
	if n1.EntityType != EntityTask || n1.EntityUUID != uuid {
		t.Errorf("entity = %v/%v", n1.EntityType, n1.EntityUUID)
	}

	// File exists with default header
	body, err := os.ReadFile(n1.Path)
	if err != nil {
		t.Fatalf("read created file: %v", err)
	}
	if !strings.Contains(string(body), "# Kickoff") {
		t.Errorf("body missing header: %q", body)
	}

	// Create another with explicit body
	n2, err := repo.Create(EntityTask, uuid, "Kickoff", "custom body")
	if err != nil {
		t.Fatalf("Create #2: %v", err)
	}
	if n2.Filename == n1.Filename {
		t.Error("collision not handled — got same filename twice")
	}
	body2, _ := os.ReadFile(n2.Path)
	if !strings.HasPrefix(string(body2), "custom body") {
		t.Errorf("custom body not written: %q", body2)
	}

	// List returns both
	notes, err = repo.List(EntityTask, uuid)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
}

func TestRepositoryListAll(t *testing.T) {
	root := t.TempDir()
	repo := NewRepository(root)

	if _, err := repo.Create(EntityTask, "task-uuid", "Task note", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Create(EntityProject, "proj-uuid", "Project note", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Create(EntityArea, "area-uuid", "Area note", ""); err != nil {
		t.Fatal(err)
	}

	all, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3, got %d", len(all))
	}

	seen := map[EntityType]bool{}
	for _, n := range all {
		seen[n.EntityType] = true
	}
	for _, et := range []EntityType{EntityTask, EntityProject, EntityArea} {
		if !seen[et] {
			t.Errorf("missing entity type %v", et)
		}
	}
}

func TestRepositorySearch(t *testing.T) {
	root := t.TempDir()
	repo := NewRepository(root)

	_, err := repo.Create(EntityTask, "u1", "Hello", "first line\nthe quick brown fox\nlast line")
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.Create(EntityTask, "u2", "Other", "nothing relevant here")
	if err != nil {
		t.Fatal(err)
	}

	matches, err := repo.Search("", "", "QUICK")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Line != 2 {
		t.Errorf("line = %d, want 2", matches[0].Line)
	}
	if !strings.Contains(matches[0].Content, "quick brown fox") {
		t.Errorf("content = %q", matches[0].Content)
	}

	// Scoped search misses other entities
	matches, err = repo.Search(EntityTask, "u2", "quick")
	if err != nil {
		t.Fatalf("scoped search: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("expected 0 in u2, got %d", len(matches))
	}
}

func TestRepositoryEntityDir(t *testing.T) {
	repo := NewRepository("/tmp/notes")
	got := repo.EntityDir(EntityProject, "abc")
	want := filepath.Join("/tmp/notes", "project", "abc")
	if got != want {
		t.Errorf("EntityDir = %q, want %q", got, want)
	}
}
