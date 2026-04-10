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

	b.WriteString("You have been activated through the TT Task Management CLI.\n\n")
	fmt.Fprintf(&b, "Context: Task #%d — %s\n", t.ID, t.Title)

	// Only include non-empty details
	if t.Description != nil && *t.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", *t.Description)
	}
	fmt.Fprintf(&b, "Status: %s | State: %s\n", t.Status, t.State)
	if t.ParentName != nil {
		fmt.Fprintf(&b, "Project: %s\n", *t.ParentName)
	}
	if t.AreaName != nil {
		fmt.Fprintf(&b, "Area: %s\n", *t.AreaName)
	}
	if t.PlannedDate != nil {
		fmt.Fprintf(&b, "Planned: %s\n", t.PlannedDate.Format("Jan 2, 2006"))
	}
	if t.DueDate != nil {
		fmt.Fprintf(&b, "Due: %s\n", t.DueDate.Format("Jan 2, 2006"))
	}
	if len(t.Tags) > 0 {
		fmt.Fprintf(&b, "Tags: %s\n", strings.Join(t.Tags, ", "))
	}

	b.WriteString("\nUse `tt --help` and `tt <command> --help` to discover available CLI commands for interacting with tasks, projects, areas, notes, and more.\n")

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
