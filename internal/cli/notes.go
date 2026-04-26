package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/devbydaniel/tt/internal/domain/note"
	noteusecases "github.com/devbydaniel/tt/internal/domain/note/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/output"
)

// NewNotesCmd builds the `tt notes` command tree.
//
// `tt notes` itself (with an entity flag) is the interactive entry point:
// it lists matching notes in fzf and opens the selection in $EDITOR.
//
// Subcommands provide non-interactive surface for scripts and agents:
//
//	tt notes ls       List notes (text or --json)
//	tt notes add      Create a note (interactive or with --title/--body)
//	tt notes search   Substring search across notes
func NewNotesCmd(deps *Dependencies) *cobra.Command {
	var taskID int64
	var projectRef string
	var areaName string

	cmd := &cobra.Command{
		Use:   "notes",
		Short: "Manage markdown notes attached to tasks, projects, and areas",
		Long: `Manage markdown notes attached to tasks, projects, and areas.

Notes are plain markdown files stored on disk under the notes directory
(default: <data_dir>/notes). To sync them across machines, run ` + "`git init`" + `
in the notes directory and add a remote — ` + "`tt sync`" + ` will then auto-commit,
pull, and push notes alongside its task sync. Syncthing/iCloud/Dropbox also
work if you'd rather not use git.

Running ` + "`tt notes --task <id>`" + ` (or --project / --area) opens an
fzf picker over the matching notes and edits the selection in $EDITOR.

For non-interactive use, see ` + "`tt notes ls`, `tt notes add`, `tt notes search`" + `.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := resolveEntityRef(deps, taskID, projectRef, areaName)
			if err != nil {
				return err
			}
			return runInteractiveNotes(deps, ref)
		},
	}
	addEntityFlags(cmd, &taskID, &projectRef, &areaName)
	registerEntityFlagCompletions(cmd, deps)

	cmd.AddCommand(newNotesListCmd(deps))
	cmd.AddCommand(newNotesBrowseCmd(deps))
	cmd.AddCommand(newNotesAddCmd(deps))
	cmd.AddCommand(newNotesSearchCmd(deps))

	return cmd
}

// ----- subcommands ----------------------------------------------------------

func newNotesListCmd(deps *Dependencies) *cobra.Command {
	var taskID int64
	var projectRef string
	var areaName string
	var jsonOutput bool
	var beforeStr string
	var afterStr string

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List notes (all, or filtered by entity)",
		Long: `List notes.

With no entity flag, lists every note across all entities.
With --task / --project / --area, lists only notes for that entity.
With --before / --after, filter notes by date (YYYY-MM-DD, inclusive).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := noteusecases.ListOptions{}

			if taskID != 0 || projectRef != "" || areaName != "" {
				ref, err := resolveEntityRef(deps, taskID, projectRef, areaName)
				if err != nil {
					return err
				}
				opts.EntityType = ref.entityType
				opts.EntityUUID = ref.entityUUID
			}

			var err error
			if opts.Before, err = noteusecases.ParseDateFlag("before", beforeStr); err != nil {
				return err
			}
			if opts.After, err = noteusecases.ParseDateFlag("after", afterStr); err != nil {
				return err
			}

			notes, err := deps.App.ListNotes.Execute(opts)
			if err != nil {
				return err
			}

			// Enrich with entity metadata for display/JSON.
			notes = enrichNotes(deps, notes)

			if jsonOutput {
				return output.WriteJSON(os.Stdout, notes)
			}
			f := output.NewFormatter(os.Stdout, nil)
			f.NoteList(notes, opts.EntityType == "")
			return nil
		},
	}
	addEntityFlags(cmd, &taskID, &projectRef, &areaName)
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&beforeStr, "before", "", "Show notes on or before this date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&afterStr, "after", "", "Show notes on or after this date (YYYY-MM-DD)")
	registerEntityFlagCompletions(cmd, deps)
	return cmd
}

func newNotesBrowseCmd(deps *Dependencies) *cobra.Command {
	var taskID int64
	var projectRef string
	var areaName string
	var beforeStr string
	var afterStr string

	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Interactively browse notes with fzf and open in $EDITOR",
		Long: `Browse notes interactively using fzf.

Works exactly like "tt notes ls" but pipes results through fzf for
fuzzy selection. The chosen note is opened in $EDITOR.

With no entity flag, browses every note across all entities.
With --task / --project / --area, browses only notes for that entity.
With --before / --after, filter notes by date (YYYY-MM-DD, inclusive).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := noteusecases.ListOptions{}
			showEntity := true

			if taskID != 0 || projectRef != "" || areaName != "" {
				ref, err := resolveEntityRef(deps, taskID, projectRef, areaName)
				if err != nil {
					return err
				}
				opts.EntityType = ref.entityType
				opts.EntityUUID = ref.entityUUID
				showEntity = false
			}

			var err error
			if opts.Before, err = noteusecases.ParseDateFlag("before", beforeStr); err != nil {
				return err
			}
			if opts.After, err = noteusecases.ParseDateFlag("after", afterStr); err != nil {
				return err
			}

			notes, err := deps.App.ListNotes.Execute(opts)
			if err != nil {
				return err
			}

			notes = enrichNotes(deps, notes)

			if len(notes) == 0 {
				fmt.Fprintln(os.Stderr, "No notes found.")
				return nil
			}

			selectedPath, err := pickNoteWithFzf(notes, showEntity)
			if err != nil {
				return err
			}
			if selectedPath == "" {
				return nil
			}
			return openInEditor(selectedPath)
		},
	}
	addEntityFlags(cmd, &taskID, &projectRef, &areaName)
	cmd.Flags().StringVar(&beforeStr, "before", "", "Show notes on or before this date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&afterStr, "after", "", "Show notes on or after this date (YYYY-MM-DD)")
	registerEntityFlagCompletions(cmd, deps)
	return cmd
}

func newNotesAddCmd(deps *Dependencies) *cobra.Command {
	var taskID int64
	var projectRef string
	var areaName string
	var title string
	var body string
	var bodyFile string

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new note",
		Long: `Create a new markdown note for a task, project, or area.

Two modes:

Interactive (default when --title is not given):
  Prompts for a title, then opens $EDITOR on the new file.

Non-interactive (when --title and --body or --body-file are given):
  Writes the file directly and prints its path. Suitable for scripts and agents.

Examples:
  tt notes add --task 5
  tt notes add --project Work --title "Kickoff"
  tt notes add --task 5 --title "Idea" --body "first draft"
  tt notes add --area Health --title "Retro" --body-file retro.md
  echo "from stdin" | tt notes add --task 5 --title "Stream" --body-file -`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := resolveEntityRef(deps, taskID, projectRef, areaName)
			if err != nil {
				return err
			}

			if body != "" && bodyFile != "" {
				return errors.New("cannot specify both --body and --body-file")
			}

			// Non-interactive path: title is required, body comes from flags.
			if title != "" || body != "" || bodyFile != "" {
				if title == "" {
					return errors.New("--title is required when --body or --body-file is set")
				}
				resolvedBody := body
				if bodyFile != "" {
					b, err := readBodyFile(bodyFile)
					if err != nil {
						return err
					}
					resolvedBody = b
				}
				n, err := deps.App.CreateNote.Execute(noteusecases.CreateOptions{
					EntityType: ref.entityType,
					EntityUUID: ref.entityUUID,
					Title:      title,
					Body:       resolvedBody,
				})
				if err != nil {
					return err
				}
				fmt.Fprintln(os.Stdout, n.Path)
				return nil
			}

			// Interactive path: prompt for title, create file, open $EDITOR.
			promptedTitle, err := promptLine(fmt.Sprintf("Note title for %s %q: ", ref.entityType, ref.displayName))
			if err != nil {
				return err
			}
			promptedTitle = strings.TrimSpace(promptedTitle)
			if promptedTitle == "" {
				return errors.New("title is required")
			}
			n, err := deps.App.CreateNote.Execute(noteusecases.CreateOptions{
				EntityType: ref.entityType,
				EntityUUID: ref.entityUUID,
				Title:      promptedTitle,
			})
			if err != nil {
				return err
			}
			if err := openInEditor(n.Path); err != nil {
				return err
			}
			fmt.Fprintln(os.Stdout, n.Path)
			return nil
		},
	}
	addEntityFlags(cmd, &taskID, &projectRef, &areaName)
	cmd.Flags().StringVar(&title, "title", "", "Note title (required for non-interactive mode)")
	cmd.Flags().StringVar(&body, "body", "", "Note body as a string")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Read note body from file (use - for stdin)")
	registerEntityFlagCompletions(cmd, deps)
	return cmd
}

func newNotesSearchCmd(deps *Dependencies) *cobra.Command {
	var taskID int64
	var projectRef string
	var areaName string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Case-insensitive substring search across notes",
		Long: `Search notes for a substring (case-insensitive).

With no entity flag, searches every note. With --task / --project / --area,
limits the search to notes for that entity.

Default output is grep-style: <path>:<line>:<content>.
Use --json for structured output.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			opts := noteusecases.SearchOptions{Query: query}
			if taskID != 0 || projectRef != "" || areaName != "" {
				ref, err := resolveEntityRef(deps, taskID, projectRef, areaName)
				if err != nil {
					return err
				}
				opts.EntityType = ref.entityType
				opts.EntityUUID = ref.entityUUID
			}

			matches, err := deps.App.SearchNotes.Execute(opts)
			if err != nil {
				return err
			}

			if jsonOutput {
				// Enrich notes inside matches with entity metadata.
				for i := range matches {
					enriched := enrichNotes(deps, []note.Note{matches[i].Note})
					if len(enriched) > 0 {
						matches[i].Note = enriched[0]
					}
				}
				return output.WriteJSON(os.Stdout, matches)
			}
			for _, m := range matches {
				fmt.Fprintf(os.Stdout, "%s:%d:%s\n", m.Note.Path, m.Line, m.Content)
			}
			return nil
		},
	}
	addEntityFlags(cmd, &taskID, &projectRef, &areaName)
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	registerEntityFlagCompletions(cmd, deps)
	return cmd
}

// ----- entity resolution ----------------------------------------------------

// entityRef is the resolved (type, uuid, displayName) for an entity flag.
type entityRef struct {
	entityType  note.EntityType
	entityUUID  string
	entityID    int64
	displayName string
}

// addEntityFlags registers --task / --project / --area on a command.
func addEntityFlags(cmd *cobra.Command, taskID *int64, projectRef, areaName *string) {
	cmd.Flags().Int64Var(taskID, "task", 0, "Task ID")
	cmd.Flags().StringVar(projectRef, "project", "", "Project name or ID")
	cmd.Flags().StringVar(areaName, "area", "", "Area name")
}

func registerEntityFlagCompletions(cmd *cobra.Command, deps *Dependencies) {
	registry := NewCompletionRegistry(deps)
	_ = cmd.RegisterFlagCompletionFunc("project", registry.AllProjectCompletion())
	_ = cmd.RegisterFlagCompletionFunc("area", registry.AreaCompletion())
}

// resolveEntityRef enforces "exactly one entity flag set" and resolves it
// against the task/area domain to a UUID.
func resolveEntityRef(deps *Dependencies, taskID int64, projectRef, areaName string) (entityRef, error) {
	count := 0
	if taskID != 0 {
		count++
	}
	if projectRef != "" {
		count++
	}
	if areaName != "" {
		count++
	}
	if count == 0 {
		return entityRef{}, errors.New("specify exactly one of --task, --project, or --area")
	}
	if count > 1 {
		return entityRef{}, errors.New("specify only one of --task, --project, or --area")
	}

	switch {
	case taskID != 0:
		t, err := deps.App.GetTask.Execute(taskID)
		if err != nil {
			return entityRef{}, err
		}
		if t.IsProject() {
			// A "task ID" that points at a project is fine — treat it as a project.
			return entityRef{
				entityType:  note.EntityProject,
				entityUUID:  t.UUID,
				entityID:    t.ID,
				displayName: t.Title,
			}, nil
		}
		return entityRef{
			entityType:  note.EntityTask,
			entityUUID:  t.UUID,
			entityID:    t.ID,
			displayName: t.Title,
		}, nil

	case projectRef != "":
		p, err := lookupProject(deps, projectRef)
		if err != nil {
			return entityRef{}, err
		}
		return entityRef{
			entityType:  note.EntityProject,
			entityUUID:  p.UUID,
			entityID:    p.ID,
			displayName: p.Title,
		}, nil

	case areaName != "":
		a, err := deps.App.GetAreaByName.Execute(areaName)
		if err != nil {
			return entityRef{}, err
		}
		return entityRef{
			entityType:  note.EntityArea,
			entityUUID:  a.UUID,
			entityID:    a.ID,
			displayName: a.Name,
		}, nil
	}

	return entityRef{}, errors.New("unreachable")
}

// lookupProject accepts either a project name or a numeric project ID.
func lookupProject(deps *Dependencies, ref string) (*task.Task, error) {
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		t, err := deps.App.GetTask.Execute(id)
		if err != nil {
			return nil, err
		}
		if !t.IsProject() {
			return nil, fmt.Errorf("task #%d is not a project", id)
		}
		return t, nil
	}
	return deps.App.GetProjectByName.Execute(ref)
}

// enrichNotes populates EntityName and EntityID on each note by looking up
// the underlying task/area by UUID. Failed lookups leave the fields empty.
func enrichNotes(deps *Dependencies, notes []note.Note) []note.Note {
	// Cache lookups by (type, uuid) to avoid hammering the DB.
	type key struct {
		t  note.EntityType
		id string
	}
	type val struct {
		name string
		id   int64
	}
	cache := make(map[key]val)

	for i := range notes {
		k := key{t: notes[i].EntityType, id: notes[i].EntityUUID}
		v, ok := cache[k]
		if !ok {
			v = lookupEntity(deps, k.t, k.id)
			cache[k] = v
		}
		notes[i].EntityName = v.name
		notes[i].EntityID = v.id
	}
	return notes
}

func lookupEntity(deps *Dependencies, et note.EntityType, entityUUID string) (v struct {
	name string
	id   int64
}) {
	switch et {
	case note.EntityTask, note.EntityProject:
		t, err := deps.App.GetTaskByUUID.Execute(entityUUID)
		if err == nil && t != nil {
			v.name = t.Title
			v.id = t.ID
		}
	case note.EntityArea:
		a, err := deps.App.GetAreaByUUID.Execute(entityUUID)
		if err == nil && a != nil {
			v.name = a.Name
			v.id = a.ID
		}
	}
	return v
}

// ----- interactive flow -----------------------------------------------------

func runInteractiveNotes(deps *Dependencies, ref entityRef) error {
	notes, err := deps.App.ListNotes.Execute(noteusecases.ListOptions{
		EntityType: ref.entityType,
		EntityUUID: ref.entityUUID,
	})
	if err != nil {
		return err
	}

	if len(notes) == 0 {
		fmt.Fprintf(os.Stderr, "No notes for %s %q.\n", ref.entityType, ref.displayName)
		fmt.Fprintf(os.Stderr, "Create one with: tt notes add --%s %s\n", ref.entityType, entityFlagValue(ref))
		return nil
	}

	selectedPath, err := pickNoteWithFzf(notes, false)
	if err != nil {
		return err
	}
	if selectedPath == "" {
		return nil // user canceled
	}
	return openInEditor(selectedPath)
}

// entityFlagValue returns the value to pass to the relevant flag.
// For tasks/projects we prefer the ID; for areas we use the name.
func entityFlagValue(ref entityRef) string {
	switch ref.entityType {
	case note.EntityArea:
		return shellQuote(ref.displayName)
	default:
		return strconv.FormatInt(ref.entityID, 10)
	}
}

func shellQuote(s string) string {
	if strings.ContainsAny(s, " \t\"'") {
		return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
	}
	return s
}

// pickNoteWithFzf displays notes in fzf and returns the absolute path of the
// selection, or "" if the user canceled. Falls back to a numeric prompt if
// fzf is not installed.
func pickNoteWithFzf(notes []note.Note, showEntity bool) (string, error) {
	if _, err := exec.LookPath("fzf"); err != nil {
		return pickNoteFallback(notes, showEntity)
	}

	// Each line is "<display>\t<path>". --with-nth=1 hides the path column.
	var input strings.Builder
	for _, n := range notes {
		var display string
		if showEntity {
			label := n.EntityName
			if label == "" {
				label = n.EntityUUID
			}
			display = fmt.Sprintf("%s  [%s] %s  %s", n.Date.Format("2006-01-02"), n.EntityType, label, n.Title)
		} else {
			display = fmt.Sprintf("%s  %s", n.Date.Format("2006-01-02"), n.Title)
		}
		input.WriteString(display)
		input.WriteByte('\t')
		input.WriteString(n.Path)
		input.WriteByte('\n')
	}

	previewCmd := "cat {2}"
	if _, err := exec.LookPath("bat"); err == nil {
		previewCmd = "bat --color=always --style=plain --language=markdown {2}"
	}

	cmd := exec.Command("fzf",
		"--height=60%",
		"--reverse",
		"--delimiter=\t",
		"--with-nth=1",
		"--preview", previewCmd,
		"--preview-window=right:60%:wrap",
		"--prompt=note> ",
	)
	cmd.Stdin = strings.NewReader(input.String())
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		// fzf exits 130 when the user cancels with Esc/Ctrl-C.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 130 {
			return "", nil
		}
		return "", fmt.Errorf("fzf: %w", err)
	}

	line := strings.TrimRight(string(out), "\n")
	if line == "" {
		return "", nil
	}
	parts := strings.SplitN(line, "\t", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected fzf output: %q", line)
	}
	return parts[1], nil
}

func pickNoteFallback(notes []note.Note, showEntity bool) (string, error) {
	fmt.Fprintln(os.Stderr, "fzf not installed — falling back to numeric picker.")
	for i, n := range notes {
		if showEntity {
			label := n.EntityName
			if label == "" {
				label = n.EntityUUID
			}
			fmt.Fprintf(os.Stderr, "  %2d  %s  [%s] %s  %s\n", i+1, n.Date.Format("2006-01-02"), n.EntityType, label, n.Title)
		} else {
			fmt.Fprintf(os.Stderr, "  %2d  %s  %s\n", i+1, n.Date.Format("2006-01-02"), n.Title)
		}
	}
	choice, err := promptLine("Select note number (empty to cancel): ")
	if err != nil {
		return "", err
	}
	choice = strings.TrimSpace(choice)
	if choice == "" {
		return "", nil
	}
	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(notes) {
		return "", fmt.Errorf("invalid selection: %q", choice)
	}
	return notes[idx-1].Path, nil
}

// ----- editor / stdio helpers -----------------------------------------------

func openInEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}
	// Allow $EDITOR like "code -w" by splitting on whitespace.
	fields := strings.Fields(editor)
	cmd := exec.Command(fields[0], append(fields[1:], path)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func promptLine(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", nil
	}
	return scanner.Text(), nil
}

func readBodyFile(path string) (string, error) {
	if path == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
