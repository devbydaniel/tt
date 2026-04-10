package tui

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/devbydaniel/tt/config"
	"github.com/devbydaniel/tt/internal/domain/task"
)

type aiSyncFinishedMsg struct {
	err error
}

// findAIBinary checks that the configured AI binary exists on PATH
func findAIBinary(cfg *config.AIConfig) (string, error) {
	path, err := exec.LookPath(cfg.Binary)
	if err != nil {
		return "", fmt.Errorf("AI binary %q not found: %w", cfg.Binary, err)
	}
	return path, nil
}

// buildAISystemPrompt constructs the --append-system-prompt content
func buildAISystemPrompt(t *task.Task) string {
	var b strings.Builder

	fmt.Fprintf(&b, "You were launched from the tt task manager in the context of task #%d: %q.\n\n", t.ID, t.Title)

	// Task details
	b.WriteString("Task details:\n")
	fmt.Fprintf(&b, "- Status: %s\n", t.Status)
	fmt.Fprintf(&b, "- State: %s\n", t.State)

	if t.Description != nil && *t.Description != "" {
		fmt.Fprintf(&b, "- Description: %s\n", *t.Description)
	} else {
		b.WriteString("- Description: None\n")
	}

	if t.ParentName != nil {
		fmt.Fprintf(&b, "- Project: %s\n", *t.ParentName)
	} else {
		b.WriteString("- Project: None\n")
	}

	if t.AreaName != nil {
		fmt.Fprintf(&b, "- Area: %s\n", *t.AreaName)
	} else {
		b.WriteString("- Area: None\n")
	}

	if t.PlannedDate != nil {
		fmt.Fprintf(&b, "- Planned: %s\n", t.PlannedDate.Format("Jan 2, 2006"))
	} else {
		b.WriteString("- Planned: None\n")
	}

	if t.DueDate != nil {
		fmt.Fprintf(&b, "- Due: %s\n", t.DueDate.Format("Jan 2, 2006"))
	} else {
		b.WriteString("- Due: None\n")
	}

	if len(t.Tags) > 0 {
		fmt.Fprintf(&b, "- Tags: %s\n", strings.Join(t.Tags, ", "))
	} else {
		b.WriteString("- Tags: None\n")
	}

	// CLI reference
	id := fmt.Sprintf("%d", t.ID)
	b.WriteString("\nYou can use the tt CLI to interact with the task system:\n")
	b.WriteString("  tt edit " + id + " --title \"...\"        # update task title\n")
	b.WriteString("  tt edit " + id + " --description \"...\"  # update description\n")
	b.WriteString("  tt list --json                        # list all tasks\n")
	b.WriteString("  tt notes ls --task " + id + "             # list notes for this task\n")
	b.WriteString("  tt notes add --task " + id + " --title \"...\" --body \"...\"  # add a note\n")
	b.WriteString("  tt do " + id + "                          # mark task complete\n")
	b.WriteString("  tt plan " + id + " <date>                 # set planned date\n")
	b.WriteString("  tt due " + id + " <date>                  # set due date\n")

	return b.String()
}

// launchAISync creates a tea.Cmd that suspends the TUI and launches the AI interactively
func launchAISync(t *task.Task, binary, workspace string) tea.Cmd {
	prompt := buildAISystemPrompt(t)
	c := exec.Command(binary, "--dangerously-skip-permissions", "--append-system-prompt", prompt)
	if workspace != "" {
		c.Dir = workspace
	}
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return aiSyncFinishedMsg{err: err}
	})
}
