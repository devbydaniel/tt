# t - CLI Task Management Tool

A Things3-inspired task manager with projects, areas, tags, and flexible recurrence.

## Data Model

### Entities

**Task**
- `id` - Short numeric ID (CLI-friendly)
- `uuid` - Internal UUID (for notes file naming)
- `title` - Task description
- `project_id` - Optional reference to project
- `planned_date` - Soft "when to work on it" (nullable)
- `due_date` - Hard deadline (nullable)
- `recurrence_rule` - Recurrence pattern (nullable)
- `state` - `active` | `someday` (visibility/scheduling)
- `status` - `todo` | `done` (completion)
- `created_at` - Timestamp
- `completed_at` - Timestamp (nullable)

**Project**
- `id` - Primary key
- `name` - Project name
- `area_id` - Optional reference to area

**Area**
- `id` - Primary key
- `name` - Area name

**TaskTag** (join table)
- `task_id` - Reference to task
- `tag_name` - Tag string (tags created implicitly)

### Notes

Notes are stored as external markdown files:
- Location: `{notes_dir}/{uuid}.md`
- Editable with any text editor
- Greppable and version-controllable

### State vs Status

Two separate concerns:

- **status** (`todo` | `done`) - Completion state. Explicit, user-controlled.
- **state** (`active` | `someday`) - Visibility/scheduling. Explicit, user-controlled.

### Derived Views

Views are computed from status + state + dates:

| View | Criteria |
|------|----------|
| Inbox | todo + active + no project + no dates |
| Today | todo + active + (planned_date = today OR overdue) |
| Upcoming | todo + active + future planned_date or due_date |
| Anytime | todo + active + has project + no dates |
| Someday | todo + someday |
| Logbook | done |

### Auto-transitions

- Adding dates to a `someday` task → automatically sets state to `active`
- `t defer` → sets state to `someday`
- All other state/status changes are explicit

## CLI Interface

Command name: `t`

### Task Commands

```
t                        # List today + overdue (default)
t list [filters]         # List tasks with filters
t add "title" [flags]    # Create task
t done <id> [id...]      # Complete task(s)
t edit <id> [flags]      # Modify task
t delete <id>            # Remove task
t show <id>              # Display task details + notes preview
t note <id>              # Open notes in $EDITOR
t log [--since DATE]     # Show completed tasks
t defer <id>             # Move task to someday
t activate <id>          # Bring task back to active
```

### List Filters

```
t list --today           # Planned today + overdue (default)
t list --upcoming        # Future planned/due tasks
t list --anytime         # Has project, no dates
t list --someday         # State = someday
t list --inbox           # No project, no dates
t list --all             # All active todos
t list --project NAME    # Filter by project
t list --area NAME       # Filter by area
t list --tag NAME        # Filter by tag
```

### Organization Commands

```
t project list
t project add <name> [--area NAME]
t project delete <name>

t area list
t area add <name>
t area delete <name>

t tag list               # Show all tags in use
```

### Configuration Commands

```
t config init            # Create default config file
t config path            # Print config file location
```

### Add/Edit Flags

```
--project, -p NAME       # Assign to project
--area, -a NAME          # Assign to area (via project or directly?)
--tag, -t NAME           # Add tag (repeatable)
--planned DATE           # Set planned date
--due, -d DATE           # Set due date
--recur RULE             # Set recurrence rule
```

## Date Input

Supports ISO dates and natural language:

- `2025-01-15` - Explicit date
- `today` - Today
- `tomorrow` - Tomorrow
- `next friday` - Next occurrence of weekday
- `+3d` - 3 days from now
- `+1w` - 1 week from now
- `+2m` - 2 months from now

## Recurrence

Two types:

1. **Fixed schedule** - Recurs on fixed dates regardless of completion
   - `every day`
   - `every 2 weeks`
   - `every monday, wednesday`
   - `every month on the 15th`

2. **Completion-based** - Recurs relative to when task is completed
   - `every 3 days after completion`
   - `every 2 weeks after completion`

When a recurring task is completed:
- Current task marked done with `completed_at`
- New task created with next occurrence date

## Configuration

Location: `~/.config/t/config.yml`

```yaml
database: ~/.local/share/t/tasks.db
notes_dir: ~/.local/share/t/notes
editor: vim  # Falls back to $EDITOR
```

## Output

- Default: Human-readable plain text
- `--json` flag: JSON output for scripting/integration

## Technical Stack

- **Language**: Go
- **Database**: SQLite
- **Config**: YAML

## Distribution

Via goreleaser:
- GitHub releases (prebuilt binaries)
- Homebrew tap
- AUR
- Scoop (Windows)

## Future Enhancements (Post-MVP)

- TUI mode for interactive browsing
- Shell completions (bash/zsh/fish)
- Color output (with `--no-color` flag)
- Import/export (JSON backup)
- Subtasks/checklists
