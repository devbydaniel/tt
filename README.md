# tt

A fast, local-first task management CLI inspired by Things 3.

Built for people who live in the terminal and want a simple, powerful way to manage tasks without leaving the command line.

## Features

- **Local-first** - All data stored in SQLite on your machine. No accounts, no sync, no cloud.
- **Single binary** - Pure Go, no dependencies. Just download and run.
- **Natural dates** - Use `tomorrow`, `friday`, `+3d`, or `2025-01-15`
- **Smart recurrence** - `every monday`, `daily`, or `3d after done`
- **Flexible organization** - Areas, projects, and tags
- **Shell completion** - Tab completion for bash, zsh, and fish
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
tt today                  # Shorthand for tt list (today's tasks + overdue)
tt upcoming               # Future planned tasks (or: tt list --upcoming)
tt someday                # Someday/maybe tasks (or: tt list --someday)
tt anytime                # Active tasks with no dates (or: tt list --anytime)
tt inbox                  # Tasks with no project, area, or dates (or: tt list --inbox)
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

All list commands support the `--group` / `-g` flag.

### Searching Tasks (`search` / `s`)

```bash
tt search "groceries"           # Search by title (case-insensitive)
tt s "report"                   # Shorthand

# Combine with list filters
tt list --search "report" --project Work
tt list -s "meeting" --upcoming
```

### Completing Tasks

```bash
tt done 1                 # Complete task #1
tt done 1 2 3             # Complete multiple tasks
```

Recurring tasks automatically create their next occurrence when completed.

### Editing Tasks (`edit` / `e`)

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

### Managing Dates (`plan` / `p`, `due` / `d`)

```bash
# Set planned date (when you want to start)
tt plan 1 tomorrow         # or: tt p 1 tomorrow
tt plan 1 monday
tt plan 1 --clear

# Set due date (when it's due)
tt due 1 friday            # or: tt d 1 friday
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

**Tags** (`tag` / `t`) - Flexible labels:

```bash
tt tag list                    # Show all tags in use
tt tag add 1 urgent            # Add tag to task (or: tt t add 1 urgent)
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

## Shell Completion

Enable tab completion for commands, flags, and dynamic values like project and area names.

### Bash

```bash
# Add to ~/.bashrc:
source <(tt completion bash)
```

### Zsh

```bash
# Add to ~/.zshrc:
source <(tt completion zsh)
```

If completion isn't working, ensure compinit is enabled:

```bash
autoload -U compinit; compinit
```

### Fish

```bash
tt completion fish > ~/.config/fish/completions/tt.fish
```

After enabling, restart your shell or source the config file. Then use Tab to complete:

```bash
tt add --project <TAB>    # Shows available projects
tt list --area <TAB>      # Shows available areas
tt edit 1 -p W<TAB>       # Completes to "Work" if it exists
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
