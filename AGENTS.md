# AGENTS.md

CLI & TUI task management

## Architecture

Layered: `cmd/` → `internal/cli/` → `internal/domain/*/` → `internal/database/`

## Adding a CLI Command

1. Create `internal/cli/<command>.go`
2. Add to `Dependencies` struct in `root.go` if new service needed
3. Register with `rootCmd.AddCommand()` in `NewRootCmd()`

## Adding a Domain Entity

1. Create `internal/domain/<entity>/model.go` - struct + constants
2. Create `repository.go` - SQL queries, takes `*database.DB`
3. Create `service.go` - business logic, takes repository
4. Wire in `cmd/t/main.go` and add to `cli.Dependencies`

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
