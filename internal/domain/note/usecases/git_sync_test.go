package usecases

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devbydaniel/tt/internal/domain/note"
)

// gitAvailable returns true if git is on PATH; tests skip otherwise so they
// remain runnable in minimal CI images.
func gitAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available on PATH")
	}
}

// setupRepo initializes a notes-style git repo at root and configures a local
// identity so commits don't fail in environments without global git config.
// If withRemote is non-empty, it's added as `origin` and the initial commit is
// pushed there with upstream tracking.
func setupRepo(t *testing.T, root, withRemote string) {
	t.Helper()
	mustGit(t, root, "init", "-b", "main")
	mustGit(t, root, "config", "user.email", "test@example.com")
	mustGit(t, root, "config", "user.name", "test")
	mustGit(t, root, "commit", "--allow-empty", "-m", "init")
	if withRemote != "" {
		mustGit(t, root, "remote", "add", "origin", withRemote)
		mustGit(t, root, "push", "-u", "origin", "main")
	}
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func newSync(root string) *SyncNotesGit {
	return &SyncNotesGit{Repo: note.NewRepository(root)}
}

func TestSyncNotesGit_SkipsWhenNotARepo(t *testing.T) {
	gitAvailable(t)
	root := t.TempDir()

	res, err := newSync(root).Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Skipped {
		t.Fatalf("expected Skipped=true, got %+v", res)
	}
}

func TestSyncNotesGit_NoChangesNoRemote(t *testing.T) {
	gitAvailable(t)
	root := t.TempDir()
	setupRepo(t, root, "")

	res, err := newSync(root).Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Skipped || res.Committed || res.Pulled || res.Pushed {
		t.Fatalf("expected all-false result, got %+v", res)
	}
}

func TestSyncNotesGit_CommitsWithoutRemote(t *testing.T) {
	gitAvailable(t)
	root := t.TempDir()
	setupRepo(t, root, "")

	if err := os.WriteFile(filepath.Join(root, "n.md"), []byte("body\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := newSync(root).Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Committed {
		t.Fatalf("expected Committed=true, got %+v", res)
	}
	if res.Pushed || res.Pulled {
		t.Fatalf("no remote, but Pushed/Pulled set: %+v", res)
	}
	if !strings.HasPrefix(res.Message, "tt: auto-sync ") {
		t.Fatalf("unexpected commit message: %q", res.Message)
	}
}

func TestSyncNotesGit_CommitsAndPushes(t *testing.T) {
	gitAvailable(t)
	bare := t.TempDir()
	mustGit(t, bare, "init", "--bare", "-b", "main")

	root := t.TempDir()
	setupRepo(t, root, bare)

	if err := os.WriteFile(filepath.Join(root, "n.md"), []byte("body\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := newSync(root).Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Committed || !res.Pushed {
		t.Fatalf("expected Committed && Pushed, got %+v", res)
	}

	// Verify the bare remote actually received the commit.
	cmd := exec.Command("git", "-C", bare, "log", "--oneline")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("inspect bare: %v", err)
	}
	if !strings.Contains(string(out), "auto-sync") {
		t.Fatalf("expected auto-sync commit in bare remote, got:\n%s", out)
	}
}

func TestSyncNotesGit_PullsRemoteChanges(t *testing.T) {
	gitAvailable(t)
	bare := t.TempDir()
	mustGit(t, bare, "init", "--bare", "-b", "main")

	// Clone A pushes a change.
	cloneA := t.TempDir()
	setupRepo(t, cloneA, bare)
	if err := os.WriteFile(filepath.Join(cloneA, "from-a.md"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, cloneA, "add", "-A")
	mustGit(t, cloneA, "commit", "-m", "add from-a")
	mustGit(t, cloneA, "push")

	// Clone B starts empty (no init commit needed — clone the bare).
	cloneB := t.TempDir()
	mustGit(t, cloneB, "clone", bare, ".")
	mustGit(t, cloneB, "config", "user.email", "test@example.com")
	mustGit(t, cloneB, "config", "user.name", "test")

	// A pushes another change after B cloned.
	if err := os.WriteFile(filepath.Join(cloneA, "from-a-2.md"), []byte("a2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, cloneA, "add", "-A")
	mustGit(t, cloneA, "commit", "-m", "add from-a-2")
	mustGit(t, cloneA, "push")

	res, err := newSync(cloneB).Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Pulled {
		t.Fatalf("expected Pulled=true, got %+v", res)
	}
	if _, err := os.Stat(filepath.Join(cloneB, "from-a-2.md")); err != nil {
		t.Fatalf("pulled file missing: %v", err)
	}
}

func TestSyncNotesGit_ErrorsOnBrokenRemote(t *testing.T) {
	gitAvailable(t)
	bare := t.TempDir()
	mustGit(t, bare, "init", "--bare", "-b", "main")

	root := t.TempDir()
	setupRepo(t, root, bare)

	// Break the remote URL.
	mustGit(t, root, "remote", "set-url", "origin", "file:///definitely/does/not/exist")

	if _, err := newSync(root).Execute(); err == nil {
		t.Fatalf("expected error from broken remote, got nil")
	}
}
