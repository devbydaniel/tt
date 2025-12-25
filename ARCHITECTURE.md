# Architecture

Based on the patterns from the announcable backend, adapted for a CLI application.

## Design Decisions

- **Local-first**: SQLite database, works offline, no server required
- **CLI + TUI in same repo**: Shared domain layer, two frontends
- **Mobile deferred**: Will be separate repo if/when needed
- **No auth**: Single-user local application

## Project Structure

```
t/
├── cmd/
│   ├── t/                      # CLI entry point
│   │   └── main.go
│   └── t-tui/                  # TUI entry point (future)
│       └── main.go
├── config/
│   └── config.go               # Configuration loading
├── internal/
│   ├── database/
│   │   ├── database.go         # SQLite connection, migrations
│   │   └── migrations/         # Embedded SQL migrations
│   ├── domain/
│   │   ├── task/
│   │   │   ├── model.go        # Task struct
│   │   │   ├── repository.go   # Data access
│   │   │   └── service.go      # Business logic
│   │   ├── project/
│   │   │   ├── model.go
│   │   │   ├── repository.go
│   │   │   └── service.go
│   │   ├── area/
│   │   │   ├── model.go
│   │   │   ├── repository.go
│   │   │   └── service.go
│   │   └── tag/
│   │       ├── model.go
│   │       ├── repository.go
│   │       └── service.go
│   ├── cli/                    # CLI commands (cobra)
│   │   ├── root.go
│   │   ├── add.go
│   │   ├── list.go
│   │   ├── done.go
│   │   ├── edit.go
│   │   ├── delete.go
│   │   ├── show.go
│   │   ├── note.go
│   │   ├── log.go
│   │   ├── defer.go
│   │   ├── activate.go
│   │   ├── project.go
│   │   ├── area.go
│   │   ├── tag.go
│   │   └── config.go
│   ├── tui/                    # TUI components (bubbletea, future)
│   │   ├── app.go
│   │   ├── list.go
│   │   └── task.go
│   ├── dateparse/
│   │   └── dateparse.go        # Natural language date parsing
│   └── output/
│       └── output.go           # Plain text and JSON formatters
├── go.mod
├── go.sum
├── Makefile
├── .goreleaser.yml
└── README.md
```

## Frontends

### CLI (`cmd/t/`)

Primary interface. Uses `internal/cli/` for cobra commands.

### TUI (`cmd/t-tui/`) - Future

Interactive terminal UI using Bubbletea. Shares all domain logic with CLI.
Both binaries built from same repo, same `go.mod`.

### Mobile - Deferred

Separate repository if/when needed. Options:
- Native apps with own SQLite (no sync)
- cr-sqlite for conflict-free sync via file storage (WebDAV, Syncthing, etc.)

## Layers

### 1. Domain Layer (`internal/domain/`)

Each domain (task, project, area, tag) follows the same structure:

**model.go** - Data structures
```go
type Task struct {
    ID            int64
    UUID          string
    Title         string
    ProjectID     *int64
    PlannedDate   *time.Time
    DueDate       *time.Time
    RecurrenceRule *string
    State         string  // active | someday
    Status        string  // todo | done
    CreatedAt     time.Time
    CompletedAt   *time.Time
}
```

**repository.go** - Data access (SQL queries)
```go
type Repository struct {
    db *database.DB
}

func NewRepository(db *database.DB) *Repository

func (r *Repository) Create(task *Task) error
func (r *Repository) GetByID(id int64) (*Task, error)
func (r *Repository) Update(task *Task) error
func (r *Repository) Delete(id int64) error
func (r *Repository) List(filter ListFilter) ([]Task, error)
```

**service.go** - Business logic
```go
type Service struct {
    repo     *Repository
    noteDir  string
}

func NewService(repo *Repository, noteDir string) *Service

func (s *Service) Create(task *Task) error
func (s *Service) Complete(id int64) error  // Handles recurrence
func (s *Service) Defer(id int64) error
func (s *Service) Activate(id int64) error
func (s *Service) OpenNote(id int64) error  // Opens $EDITOR
```

### 2. CLI Layer (`internal/cli/`)

CLI commands using cobra. Each command receives dependencies:

```go
type Dependencies struct {
    TaskService    *task.Service
    ProjectService *project.Service
    AreaService    *area.Service
    TagService     *tag.Service
    Config         *config.Config
}

func NewAddCmd(deps *Dependencies) *cobra.Command {
    return &cobra.Command{
        Use:   "add",
        Short: "Add a new task",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Parse flags, call service, format output
        },
    }
}
```

### 3. Database Layer (`internal/database/`)

SQLite with embedded migrations:

```go
type DB struct {
    conn *sql.DB
}

func Open(path string) (*DB, error)
func (db *DB) Migrate() error
func (db *DB) Close() error

// Transaction support
func (db *DB) BeginTx() (*Tx, error)
func (tx *Tx) Commit() error
func (tx *Tx) Rollback() error
```

Migrations embedded using `embed.FS`:
```go
//go:embed migrations/*.sql
var migrations embed.FS
```

### 4. Configuration (`config/`)

YAML-based configuration:

```go
type Config struct {
    Database string `yaml:"database"`
    NotesDir string `yaml:"notes_dir"`
    Editor   string `yaml:"editor"`
}

func Load() (*Config, error)
func (c *Config) Path() string
func Default() *Config
```

Config resolution:
1. `~/.config/t/config.yml`
2. Environment variables (`T_DATABASE`, `T_NOTES_DIR`, `T_EDITOR`)
3. Defaults

## Dependency Injection

Dependencies are wired in `main.go`:

```go
func main() {
    cfg := config.Load()

    db, err := database.Open(cfg.Database)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    if err := db.Migrate(); err != nil {
        log.Fatal(err)
    }

    deps := &cmd.Dependencies{
        TaskService:    task.NewService(task.NewRepository(db), cfg.NotesDir),
        ProjectService: project.NewService(project.NewRepository(db)),
        AreaService:    area.NewService(area.NewRepository(db)),
        TagService:     tag.NewService(tag.NewRepository(db)),
        Config:         cfg,
    }

    rootCmd := cmd.NewRootCmd(deps)
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

## Error Handling

- Functions return `error` as last value
- Commands print user-friendly messages to stderr
- Exit with non-zero code on error
- `--json` flag outputs structured error objects

```go
func (c *addCmd) Run() error {
    task, err := c.deps.TaskService.Create(...)
    if err != nil {
        if errors.Is(err, project.ErrNotFound) {
            return fmt.Errorf("project %q not found", c.project)
        }
        return err
    }
    // ...
}
```

## Output Formatting

`internal/output/` handles both plain text and JSON:

```go
type Formatter interface {
    Task(t *task.Task)
    TaskList(tasks []task.Task)
    Project(p *project.Project)
    Error(err error)
}

func NewFormatter(json bool, w io.Writer) Formatter
```

## Key Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/charmbracelet/bubbletea` - TUI framework (future)
- `modernc.org/sqlite` - Pure Go SQLite (no CGO)
- `gopkg.in/yaml.v3` - YAML config parsing
- `github.com/google/uuid` - UUID generation

## Database Schema

```sql
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    project_id INTEGER REFERENCES projects(id),
    planned_date TEXT,
    due_date TEXT,
    recurrence_rule TEXT,
    state TEXT NOT NULL DEFAULT 'active',
    status TEXT NOT NULL DEFAULT 'todo',
    created_at TEXT NOT NULL,
    completed_at TEXT
);

CREATE TABLE projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    area_id INTEGER REFERENCES areas(id)
);

CREATE TABLE areas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE task_tags (
    task_id INTEGER REFERENCES tasks(id) ON DELETE CASCADE,
    tag_name TEXT NOT NULL,
    PRIMARY KEY (task_id, tag_name)
);

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_state ON tasks(state);
CREATE INDEX idx_tasks_planned_date ON tasks(planned_date);
CREATE INDEX idx_tasks_due_date ON tasks(due_date);
CREATE INDEX idx_tasks_project_id ON tasks(project_id);
```

## Testing Strategy

- Unit tests for services (mock repositories)
- Integration tests for repositories (in-memory SQLite)
- CLI tests using `cobra`'s test helpers
- Table-driven tests for date parsing

```go
func TestTaskService_Complete(t *testing.T) {
    repo := &mockRepository{}
    svc := task.NewService(repo, "/tmp/notes")

    // Test completion creates new task for recurring
    // ...
}
```

## Future: Multi-Device Sync

If sync is needed later, consider [cr-sqlite](https://github.com/vlcn-io/cr-sqlite):

- Adds CRDT-based conflict resolution to SQLite
- Each device maintains local database
- Changes merge without conflicts
- Works with any file transport (WebDAV, Syncthing, Dropbox, iCloud)

This keeps the local-first architecture while enabling sync without a server.
