package usecases

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/devbydaniel/tt/internal/domain/note"
)

// GitSyncResult summarizes what a single SyncNotesGit run did.
type GitSyncResult struct {
	Skipped   bool   // notes directory is not a git repo
	Committed bool   // local changes were staged and committed
	Pulled    bool   // git pull --rebase moved HEAD
	Pushed    bool   // git push ran (only when a remote is configured)
	Message   string // commit message used (empty if nothing committed)
}

// SyncNotesGit auto-commits local note changes and synchronizes the notes
// directory with its configured git remote. It is a no-op when the notes
// directory is not a git repository — users opt in by running `git init`
// (and adding a remote) themselves.
type SyncNotesGit struct {
	Repo *note.Repository
}

func (s *SyncNotesGit) Execute() (*GitSyncResult, error) {
	root := s.Repo.Root()
	result := &GitSyncResult{}

	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			result.Skipped = true
			return result, nil
		}
		return nil, fmt.Errorf("checking notes git repo: %w", err)
	}

	if _, err := runGit(root, "add", "-A"); err != nil {
		return nil, err
	}

	staged, err := hasStagedChanges(root)
	if err != nil {
		return nil, err
	}
	if staged {
		msg := fmt.Sprintf("tt: auto-sync %s", time.Now().Format("2006-01-02 15:04"))
		if _, err := runGit(root, "commit", "-m", msg); err != nil {
			return nil, err
		}
		result.Committed = true
		result.Message = msg
	}

	// Without a remote there is nothing more to do — local versioning only.
	remotes, err := runGit(root, "remote")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(remotes)) == "" {
		return result, nil
	}

	beforePull, err := runGit(root, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	if _, err := runGit(root, "pull", "--rebase"); err != nil {
		return nil, err
	}
	afterPull, err := runGit(root, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(beforePull)) != strings.TrimSpace(string(afterPull)) {
		result.Pulled = true
	}

	if _, err := runGit(root, "push"); err != nil {
		return nil, err
	}
	result.Pushed = true

	return result, nil
}

func runGit(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("git %s: %w\n%s",
			strings.Join(args, " "), err, strings.TrimRight(string(out), "\n"))
	}
	return out, nil
}

// hasStagedChanges returns true if there is anything staged for commit.
// `git diff --cached --quiet` exits 0 when clean, 1 when there is a diff.
func hasStagedChanges(dir string) (bool, error) {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = dir
	err := cmd.Run()
	if err == nil {
		return false, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return true, nil
	}
	return false, fmt.Errorf("git diff --cached --quiet: %w", err)
}
