# AGENTS.md

CLI & TUI task management

## Architecture

Layered: `cmd/` → `internal/cli/` → `internal/app/` → `internal/domain/*/usecases/` → `internal/database/`

- **Use case-based**: Each operation is its own struct with focused dependencies
- **Consumer-defined interfaces**: Cross-domain deps use interfaces defined by the consumer
- **App wiring**: `internal/app/app.go` creates all use cases and resolves dependencies

## Adding a CLI Command

1. Create `internal/cli/<command>.go`
2. Use `deps.App.<UseCase>.Execute()` to call business logic
3. Register with `rootCmd.AddCommand()` in `NewRootCmd()`

## Adding a Domain Entity

1. Create `internal/domain/<entity>/model.go` - struct + constants
2. Create `repository.go` - SQL queries, takes `*database.DB`
3. Create `usecases/` directory with one file per operation:
   - Define interfaces for cross-domain dependencies in the use case file
   - Each use case is a struct with an `Execute()` method
4. Wire use cases in `internal/app/app.go`

## Adding a Use Case

1. Create `internal/domain/<entity>/usecases/<action>.go`
2. Define any needed interfaces (e.g., `ProjectLookup`, `AreaLookup`)
3. Create struct with `Execute()` method
4. Add to `App` struct and wire in `internal/app/app.go`

Example:
```go
// internal/domain/task/usecases/create.go
type ProjectLookup interface {
    Execute(name string) (*project.Project, error)
}

type CreateTask struct {
    Repo          *task.Repository
    ProjectLookup ProjectLookup
}

func (c *CreateTask) Execute(title string, opts *CreateOptions) (*task.Task, error) {
    // business logic
}
```

## Database

- Migrations: `internal/database/migrations/NNN_name.sql`
- Auto-run on startup via `db.Migrate()`
- Uses `modernc.org/sqlite` (pure Go, no CGO)

## Output

- Use `internal/output/Formatter` for all user-facing output
- Add new methods to formatter, don't print directly in commands

## Testing

- Test functionality, not coverage. Focus on business logic and edge cases.
- Use `internal/testutil.NewTestDB(t)` for in-memory SQLite with migrations

## Development

- Use `make dev` to run with a local dev database (`./dev-data/tasks.db`)
- Set `TT_DATA_DIR` env var to use a custom database directory
- Config file: `~/.config/tt/config.toml` with `data_dir = "/path"`
- Priority: env var > config file > default (`~/.local/share/tt/`)
