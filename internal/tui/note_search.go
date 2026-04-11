package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// launchNoteSearchScope suspends the TUI and opens fzf to search notes in the
// given directory. When the user selects a note, it opens in $EDITOR. On exit,
// a scopeNoteEditorFinishedMsg is returned so notes are reloaded.
func launchNoteSearchScope(notesDir string) tea.Cmd {
	editorBin, editorArgs := resolveEditor()
	editorCmd := editorBin
	if len(editorArgs) > 0 {
		editorCmd = editorBin + " " + strings.Join(editorArgs, " ")
	}
	preview := resolvePreviewCmd()
	script := fmt.Sprintf(
		`selected=$(find %s -name '*.md' -type f 2>/dev/null | sort -r | fzf --preview '%s {}' --preview-window=right:60%%:wrap) && %s "$selected"`,
		shellQuote(notesDir), preview, editorCmd,
	)
	c := exec.Command("sh", "-c", script)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return scopeNoteEditorFinishedMsg{err: ignoreExit130(err)}
	})
}

// launchNoteSearchTask is the same as launchNoteSearchScope but returns a
// noteEditorFinishedMsg so the detail pane reloads the task's notes.
func launchNoteSearchTask(notesDir string, taskID int64, taskUUID string) tea.Cmd {
	editorBin, editorArgs := resolveEditor()
	editorCmd := editorBin
	if len(editorArgs) > 0 {
		editorCmd = editorBin + " " + strings.Join(editorArgs, " ")
	}
	preview := resolvePreviewCmd()
	script := fmt.Sprintf(
		`selected=$(find %s -name '*.md' -type f 2>/dev/null | sort -r | fzf --preview '%s {}' --preview-window=right:60%%:wrap) && %s "$selected"`,
		shellQuote(notesDir), preview, editorCmd,
	)
	c := exec.Command("sh", "-c", script)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return noteEditorFinishedMsg{taskID: taskID, taskUUID: taskUUID, err: ignoreExit130(err)}
	})
}

// resolveEditor returns the user's preferred editor binary and any extra
// arguments. It splits $EDITOR / $VISUAL on whitespace so that values like
// "code --wait" work correctly.
func resolveEditor() (string, []string) {
	raw := os.Getenv("EDITOR")
	if raw == "" {
		raw = os.Getenv("VISUAL")
	}
	if raw == "" {
		raw = "vi"
	}
	fields := strings.Fields(raw)
	return fields[0], fields[1:]
}

// resolvePreviewCmd returns "bat" with styling flags if available, otherwise "cat".
func resolvePreviewCmd() string {
	if _, err := exec.LookPath("bat"); err == nil {
		return "bat --style=plain --color=always --language=md"
	}
	return "cat"
}

// shellQuote wraps a string in single quotes for safe shell interpolation.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// ignoreExit130 returns nil for exit code 130 (user canceled fzf with Esc/Ctrl-C).
func ignoreExit130(err error) error {
	if err == nil {
		return nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
		return nil
	}
	return err
}
