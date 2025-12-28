# tt

A fast, local-first task management CLI inspired by Things 3.

Built for people who live in the terminal and want a simple, powerful way to manage tasks without leaving the command line.

## Features

- **Local-first** - All data stored in SQLite on your machine. No accounts, no sync, no cloud.
- **Single binary** - Pure Go, no dependencies. Just download and run.
- **Natural dates** - Use `tomorrow`, `friday`, `+3d`, or `2025-01-15`
- **Smart recurrence** - `every monday`, `daily`, or `3d after done`
- **Flexible organization** - Areas, projects, and tags
- **Fast** - Instant startup, instant results

## Installation

### From Source

```bash
git clone https://github.com/yourusername/tt.git
cd tt
make build
```

This creates the `tt` binary in the current directory. Move it somewhere in your PATH:

```bash
mv tt /usr/local/bin/
```

### Requirements

- Go 1.21 or later (for building from source)

## Quick Start

```bash
# Add your first task
tt add "Buy groceries"

# Add a task with a due date
tt add "Submit report" --due friday

# See today's tasks
tt

# Mark a task done
tt done 1

# See all commands
tt --help
```

## Usage

**Command aliases** for faster typing:

| Command | Alias |
|---------|-------|
| `edit`  | `e`   |
| `plan`  | `p`   |
| `due`   | `d`   |
| `tag`   | `t`   |

### Adding Tasks

```bash
tt add "Task title"
tt add "Task title" --due tomorrow
tt add "Task title" --project Work --tag urgent
tt add "Task title" --planned "+3d" --due "+1w"
tt add "Task title" -d "More details about this task"
tt add "Task title" -a Work -T            # Plan for today
tt add "Someday task" --someday
```

**Flags:**
- `--description, -d` - Task description
- `--due, -D` - Due date
- `--planned, -P` - Planned/start date
- `--today, -T` - Set planned date to today
- `--project, -p` - Assign to project
- `--area, -a` - Assign to area
- `--tag, -t` - Add tag (can be used multiple times)
- `--recur, -r` - Recurrence pattern
- `--someday` - Mark as someday/maybe

### Listing Tasks

```bash
tt                        # Today's tasks + overdue (default)
tt list                   # Same as above
tt list --upcoming        # Future planned tasks
tt list --someday         # Someday/maybe tasks
tt list --anytime         # Active tasks with no dates
tt list --inbox           # Tasks with no project, area, or dates
tt list --all             # All incomplete tasks

# Filter by organization
tt list --project Work
tt list --area Health
tt list --tag urgent

# Group output
tt list --all --group=project   # Group by Area > Project
tt list --all --group=area      # Group by area
tt list --all --group=date      # Group by date (Overdue, Today, Tomorrow, etc.)
tt list --all --group=none      # Flat list (default)
```

**Shorthand commands** for quick access:

```bash
tt inbox                  # Tasks with no project, area, or dates
tt today                  # Today's tasks + overdue
tt upcoming               # Future planned tasks
tt anytime                # Active tasks with no dates
tt someday                # Someday/maybe tasks
```

All list commands support the `--group` / `-g` flag.

### Completing Tasks

```bash
tt done 1                 # Complete task #1
tt done 1 2 3             # Complete multiple tasks
```

Recurring tasks automatically create their next occurrence when completed.

### Editing Tasks

```bash
tt edit 1                          # View task details
tt edit 1 --title "New title"
tt edit 1 -d "Add a description"
tt edit 1 --due friday
tt edit 1 --today                  # Plan for today
tt edit 1 --project Work
tt edit 1 --tag important
tt edit 1 --untag old-tag
tt edit 1 --clear-due
tt edit 1 --clear-project
tt edit 1 --clear-description

# Edit multiple tasks at once
tt edit 1 2 3 --project Work
tt edit 1 2 3 --tag urgent
tt edit 1 2 3 -T                   # Plan all for today
```

### Managing Dates

```bash
# Set planned date (when you want to start)
tt plan 1 tomorrow
tt plan 1 monday
tt plan 1 --clear

# Set due date (when it's due)
tt due 1 friday
tt due 1 +1w
tt due 1 --clear
```

**Supported date formats:**
- Keywords: `today`, `tomorrow`
- Weekdays: `monday`, `friday`, `next tuesday`
- Relative: `+3d` (3 days), `+1w` (1 week), `+2m` (2 months)
- ISO: `2025-01-15`

### Recurrence

```bash
# Set recurrence when creating
tt add "Daily standup" --recur daily
tt add "Weekly review" --recur "every friday"

# Manage recurrence on existing tasks
tt recur 1 daily
tt recur 1 "every monday"
tt recur 1 "every 2 weeks"
tt recur 1 "3d after done"      # Relative to completion
tt recur 1 --clear
tt recur 1 --pause
tt recur 1 --resume
tt recur 1 --show               # Show recurrence details
```

**Recurrence patterns:**
- Fixed: `daily`, `weekly`, `monthly`, `every monday`, `every 2 weeks`
- Relative: `3d after done`, `1w after done` (creates next task N days/weeks after completion)

### Organization

**Areas** - High-level life categories:

```bash
tt area list
tt area add Work
tt area add Health
tt area delete Work
```

**Projects** - Groups of related tasks:

```bash
tt project list
tt project add "Q1 Goals"
tt project add "Home Renovation" --area Home
tt project delete "Q1 Goals"
```

**Tags** - Flexible labels:

```bash
tt tag list                    # Show all tags in use
tt tag add 1 urgent            # Add tag to task
tt tag remove 1 urgent         # Remove tag from task
```

### Viewing Completed Tasks

```bash
tt log                         # Recent completed tasks
tt log --since 2025-01-01      # Since specific date
```

### Deleting Tasks

```bash
tt delete 1
tt delete 1 2 3
```

## Configuration

Configuration file location: `~/.config/tt/config.toml` (or `$XDG_CONFIG_HOME/tt/config.toml`)

```toml
# Custom data directory (optional)
data_dir = "/path/to/data"

# Default view for `tt` and `tt list` commands
# Options: today, upcoming, anytime, someday, inbox, all
default_list = "today"

# Default grouping for list commands
[grouping]
default = "project"    # Global default: project, area, date, none
today = "project"      # Override for today command
upcoming = "date"      # Override for upcoming command
log = "date"           # Override for log command
```

The `--group` flag always overrides config settings. View flags like `--today` or `--upcoming` override `default_list`.

## Data Storage

Your tasks are stored in a local SQLite database:

- **Default**: `~/.local/share/tt/tasks.db`
- **With XDG**: `$XDG_DATA_HOME/tt/tasks.db`
- **With config**: Path specified in `data_dir`
- **With env var**: `$TT_DATA_DIR/tasks.db`

Priority: env var > config file > default

The database is created automatically on first run.

## Building

```bash
make build    # Build the binary
make test     # Run tests
make clean    # Remove binary
```

## Architecture

```
Areas (e.g., "Work", "Health")
└── Projects (e.g., "Q1 Goals")
    └── Tasks
        └── Tags (flat, multiple per task)
```

Tasks have two independent dimensions:
- **Status**: `todo` or `done`
- **State**: `active` or `someday`

This lets you filter between "what I'm working on" and "what I'm not ready for yet."

## License

MIT

## Contributing

Contributions welcome! Please open an issue to discuss significant changes before submitting a PR.
