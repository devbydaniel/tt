# tt

A fast, local-first task management CLI inspired by Things 3.

Built for people who live in the terminal and want a simple, powerful way to manage tasks without leaving the command line.

## Features

- **Local-first** - All data stored in SQLite on your machine. No accounts, no sync, no cloud.
- **Single binary** - Pure Go, no dependencies. Just download and run.
- **Natural dates** - Use `tomorrow`, `friday`, `+3d`, or `2025-01-15`
- **Smart recurrence** - `every monday`, `daily`, or `3d after done`
- **Flexible organization** - Areas, projects, and tags
- **Interactive TUI** - Full-featured terminal UI with vim-style navigation
- **Shell completion** - Tab completion for bash, zsh, and fish
- **Themeable** - Preset themes (Dracula, Nord, etc.) or custom colors
- **Fast** - Instant startup, instant results

## Installation

### From Source

```bash
git clone https://github.com/devbydaniel/tt.git
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
# Launch the interactive TUI
tt

# Or use CLI commands directly:
tt add "Buy groceries"
tt add "Submit report" --due friday
tt today                  # See today's tasks
tt do 1                   # Mark a task done
tt --help                 # See all commands
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
- `--recur-end` - Recurrence end date
- `--someday` - Mark as someday/maybe

### Listing Tasks

```bash
tt list                   # All incomplete tasks
tt today                  # Today's tasks + overdue
tt upcoming               # Future planned tasks (or: tt list --upcoming)
tt someday                # Someday/maybe tasks (or: tt list --someday)
tt anytime                # Tasks with no dates but with a project/area (or: tt list --anytime)
tt inbox                  # Tasks with no project, area, or dates (or: tt list --inbox)

# Filter (with tab completion)
tt list --project Work
tt list --area Health
tt list --tag urgent

# Group output
tt list --group=schedule  # Group by schedule (Today, Upcoming, Anytime, Someday)
tt list --group=scope     # Group by scope (Area, Area > Project, or Project)
tt list --group=date      # Group by date (Overdue, Today, Tomorrow, etc.)
tt list --group=none      # Flat list (default)

# Filter by project/area
tt list --project "Backend API"
tt list --project "Backend API" -g schedule  # Group by schedule
tt list --project "Backend API" --hide-scope # Hide redundant project column
```

All list commands support the `--group` / `-g` flag.

### Sorting Tasks

```bash
tt list -s id                   # Sort by ID (ascending, default)
tt list --sort created          # Sort by creation date (newest first)
tt list -s title                # Sort alphabetically by title
tt list -s due                  # Sort by due date (newest first)
tt list -s planned:asc          # Sort by planned date (oldest first)
tt list -s due,title            # Multi-field: by due date, then title
tt list -s project:asc,title    # By project name, then title
```

**Sort fields:** `id`, `title`, `planned`, `due`, `created`, `project`, `area`

**Defaults:**

- Default sort is `id:asc` (oldest first) unless configured otherwise
- Date fields (`planned`, `due`, `created`) default to descending (newest first)
- Other fields (`id`, `title`, `project`, `area`) default to ascending
- Tasks without values (e.g., no due date) always sort last

### Searching Tasks (`search` / `s`)

```bash
tt search "groceries"           # Search by title (case-insensitive)
tt s "report"                   # Shorthand

# Combine with list filters
tt list --search "report" --project Work
tt list -S "meeting" --upcoming
```

### Completing Tasks

```bash
tt do 1                   # Complete task #1
tt do 1 2 3               # Complete multiple tasks
```

Recurring tasks automatically create their next occurrence when completed.

### Uncompleting Tasks

```bash
tt undo 1                 # Mark task #1 as not complete
tt undo 1 2 3             # Uncomplete multiple tasks
```

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
tt edit 1 --someday                # Move to someday
tt edit 1 --active                 # Move back to active

# Edit multiple tasks at once
tt edit 1 2 3 --project Work
tt edit 1 2 3 --tag urgent
tt edit 1 2 3 -T                   # Plan all for today

# Rename shortcut
tt rename 1 "New title"            # Shortcut for edit --title
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
tt area rename Work Business
tt area delete Work
```

**Projects** - Groups of related tasks:

```bash
tt project list
tt project list --group area    # Group by area
tt project list -g area         # Shorthand
tt project add "Q1 Goals"
tt project add "Home Renovation" --area Home
tt project rename "Q1 Goals" "Q1 Objectives"
tt project move "Home Renovation" --area Personal
tt project move "Home Renovation" --clear      # Remove from area
tt project edit "Q1 Goals" --someday           # Move project to someday
tt project edit "Q1 Goals" --active            # Move project back to active
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

### Notes

Attach markdown notes to any task, project, or area. Notes live as plain
`.md` files on disk under the notes directory (default: `<data_dir>/notes`),
organized as `<entity_type>/<entity_uuid>/YYYYMMDD--<slug>.md`.

Notes are **not** synced with the task database. If you want them on multiple
devices, point Syncthing/iCloud/Dropbox at the notes directory.

**Interactive (fzf + `$EDITOR`)**

```bash
tt notes --task 5              # fzf-pick a note for task #5, edit in $EDITOR
tt notes --project Work        # same, by project name
tt notes --project 12          # same, by project ID
tt notes --area Health         # same, by area name

tt notes add --task 5          # prompt for title, then open $EDITOR on the new file
tt notes add --project Work
tt notes add --area Health
```

If `bat` is installed, the fzf preview uses it for syntax-highlighted markdown.
If `fzf` isn't installed, `tt notes --task <id>` falls back to a numeric picker.

**Non-interactive (scripts and AI agents)**

```bash
# List notes — text or JSON, optionally filtered by entity
tt notes ls                            # all notes everywhere
tt notes ls --task 5
tt notes ls --project Work --json

# Browse notes interactively with fzf, open selection in $EDITOR
tt notes browse                        # all notes everywhere
tt notes browse --task 5               # scoped to one entity
tt notes browse --after 2025-01-01     # date-filtered

# Create a note non-interactively (prints the path)
tt notes add --task 5 --title "Kickoff" --body "first thoughts"
tt notes add --area Health --title "Retro" --body-file retro.md
echo "from stdin" | tt notes add --task 5 --title "Stream" --body-file -

# Substring search across notes (case-insensitive)
tt notes search "kickoff"              # all notes
tt notes search "kickoff" --task 5     # scoped to one entity
tt notes search "kickoff" --json       # structured matches
```

Reading and editing existing notes by path is left to plain Unix tools — `cat`,
`bat`, `vim`, `rg`, etc. Once you have a path from `tt notes ls`, you don't
need `tt` to operate on it.

**Configuration**

```toml
# Override the notes directory in ~/.config/tt/config.toml
notes_dir = "~/Documents/tt-notes"
```

Or via environment: `TT_NOTES_DIR=/path/to/notes`.

### Interactive TUI

Running `tt` without arguments launches the interactive terminal UI:

```bash
tt
```

#### Layout

The TUI has three panes with a context-sensitive help bar at the bottom:

- **Sidebar** (left) - Navigate between views (Inbox, Today, Upcoming, Anytime, Someday) and your areas/projects/tags
- **Content** (center) - View tasks or notes for the selected filter, toggle with `Tab`
- **Detail pane** (right) - Inspect and edit the selected task, toggle between data fields and notes with `Tab`

Each pane has a header showing the current view title (left-aligned) and available tabs (right-aligned).

#### Navigation

| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Move up/down |
| `h/l` | Switch panes left/right |
| `Tab` / `Shift+Tab` | Context-dependent: cycle sidebar sections, toggle Tasks/Notes in content, or cycle detail views |
| `Enter` | Select / edit field |
| `Esc` | Go back / close |
| `q` | Quit |

#### Sidebar

The sidebar has three full-height sections you can cycle through with `Tab`:

1. **Lists** - Inbox, Today, Upcoming, Anytime, Someday
2. **Scopes** - Your areas and projects (hierarchical, projects nested under areas)
3. **Tags** - All tags in use (prefixed with `#`)

**Creating items:**
- `a` - Add new task (from Lists section), or add new project (from Scopes section)
- `A` - Add new area (from Scopes section)

**When a project is selected:**
| Key | Action |
|-----|--------|
| `Space` | Complete project (and all tasks) |
| `r` | Rename project |
| `m` | Move to different area |
| `s` | Toggle someday/active |
| `Backspace` | Delete project |

**When an area is selected:**
| Key | Action |
|-----|--------|
| `r` | Rename area |
| `Backspace` | Delete area |

#### Content Pane

The content pane has two views, toggled with `Tab`:

**Tasks view** (default):

| Key | Action |
|-----|--------|
| `Space` | Mark done/undone |
| `r` | Rename task |
| `m` | Move to project/area |
| `p` | Set planned date |
| `d` | Set due date |
| `t` | Edit tags |
| `s` | Toggle someday/active |
| `a` | Add new task |
| `!` | Launch AI sync for selected task |
| `Backspace` | Delete task |
| `Enter` or `l` | Open detail pane |

Tasks display inline indicators: `★` for planned today, `⚑` for due/overdue, and `¶` for tasks with notes attached.

**Notes view** (press `Tab`):

Shows scope-level notes for the selected project or area.

| Key | Action |
|-----|--------|
| `a` | Create new note |
| `s` | Search notes with fzf |
| `l` | Preview note with syntax-highlighted markdown |
| `Enter` | Open note in `$EDITOR` |
| `Backspace` | Delete note |

In the preview pane, use `j/k` to scroll, `Enter` to open in `$EDITOR`, and `Esc` to close.

#### Detail Pane

The detail pane has two view modes, toggled with `Tab`:

**Data view** - Editable fields for the selected task:

- **Title** - Task name
- **Description** - Multi-line text
- **Scope** - Project/area assignment
- **Planned** - Start date
- **Due** - Due date
- **Tags** - Associated tags

Navigate with `j/k` or `Tab`/`Shift+Tab` and press `Enter` to edit any field.

**Notes view** - Notes attached to the selected task:

| Key | Action |
|-----|--------|
| `a` | Create new note for this task |
| `s` | Search notes with fzf |
| `Enter` | Open note in `$EDITOR` |
| `Backspace` | Delete note |

#### AI Sync

Press `!` on any task to launch an AI assistant (e.g., Claude Code) with the task's full context — title, description, dates, tags, project, and area. The AI can modify the task using `tt` CLI commands and returns to the TUI when done.

Configure the AI binary in `~/.config/tt/config.toml`.

#### Modals

**Add Task** - Multi-field form. Use `Tab` to move between fields, `Enter` to submit, `Esc` to cancel. Pre-fills scope when invoked from a project/area view.

**Date Picker** - Type natural dates (e.g., `tomorrow`, `+3d`, `friday`) or press `Tab` to switch to a calendar picker. Use arrow keys to navigate the calendar.

**Move** - Searchable list of projects and areas. Type to fuzzy-filter, `Enter` to select.

**Tags** - Toggle tags with `Space`, type to filter existing tags or create new ones.

**Description** - Multi-line text editor. Save with `Ctrl+S` or `Alt+Enter`.

**Create Note** - Enter a title and the note opens immediately in `$EDITOR`.

The TUI respects your theme and sort/group settings from the config file.

## Configuration

Configuration file location: `~/.config/tt/config.toml` (or `$XDG_CONFIG_HOME/tt/config.toml`)

```toml
# Custom data directory (optional)
data_dir = "/path/to/data"

# Global defaults for all list views
sort = "created"       # created, title, planned, due, id, project, area
group = "scope"        # scope, date, none

# Per-list overrides
[today]
sort = "planned"
group = "scope"

[upcoming]
sort = "planned:asc"
group = "date"

[project]
hide_scope = true      # Hide project/area columns when filtering by project

[area]
hide_scope = true      # Hide project/area columns when filtering by area

[tag]
hide_scope = true      # Hide project/area columns when filtering by tag

[inbox]
group = "none"

[list]
group = "scope"        # Settings for the default "tt list" view

[log]
group = "date"

[project_list]
group = "area"         # area or none
```

The `--sort` and `--group` flags always override config settings.

### Theming

Customize colors and icons to match your terminal theme:

```toml
[theme]
name = "dracula"  # Use a preset theme
```

**Available presets:**

| Theme              | Type  |
| ------------------ | ----- |
| `dracula`          | Dark  |
| `nord`             | Dark  |
| `gruvbox`          | Dark  |
| `tokyo-night`      | Dark  |
| `kanagawa`         | Dark  |
| `houston`          | Dark  |
| `solarized-light`  | Light |
| `catppuccin-latte` | Light |

**Custom colors:**

```toml
[theme]
# Colors: ANSI codes (0-255) or hex (#RRGGBB)
muted = "#6272a4"    # Dates, tags, secondary info
accent = "#f1fa8c"   # Planned-today indicator (★)
warning = "#ff5555"  # Due/overdue indicator (⚑)
success = "#50fa7b"  # Success messages
error = "#ff5555"    # Error messages
header = "#bd93f9"   # Section headers
id = "#6272a4"       # Task IDs (defaults to muted if empty)
scope = "#8be9fd"    # Project/area column
```

**Custom icons:**

```toml
[theme.icons]
planned = "★"   # Tasks planned for today
due = "⚑"       # Due/overdue indicator
date = "›"      # Planned date prefix
done = "✓"      # Completed tasks indicator
```

You can combine a preset with custom overrides - preset colors are applied first, then your custom values override them.

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

## Sync Server

The `tt-sync` server enables syncing tasks between multiple devices using event-based synchronization.

### Server Setup

**Build the server:**

```bash
make build-sync    # Creates tt-sync binary
```

**Configure the server:**

The server requires an API key for authentication. Set it via environment variable or `.env` file:

```bash
# Option 1: Copy the example and configure
cp .env.example .env
# Edit .env and set TT_SYNC_API_KEY=your-secret-key

# Option 2: Set environment variable directly
export TT_SYNC_API_KEY=your-secret-key
```

| Variable | Required | Description |
|----------|----------|-------------|
| `TT_SYNC_API_KEY` | Yes | Shared secret for authentication |
| `TT_DATA_DIR` | No | Data directory (default: `~/.local/share/tt`) |
| `PORT` | No | HTTP port (default: `8080`) |

**Run the server:**

```bash
make run-sync       # Build and run
# or
./tt-sync           # Run directly after building
```

**Docker:**

```bash
docker run -e TT_SYNC_API_KEY=your-secret -e TT_DATA_DIR=/data -v ./data:/data -p 8080:8080 tt-sync
```

### Client Configuration

Configure each client to connect to the sync server in `~/.config/tt/config.toml`:

```toml
# Unique identifier for this device (required for sync)
client_id = "macbook"  # e.g., "macbook", "work-pc", "phone"

[sync]
url = "http://localhost:8080"    # Your sync server URL
api_key = "your-secret-key"      # Must match server's TT_SYNC_API_KEY
```

Or use environment variables:

```bash
export TT_CLIENT_ID=macbook
export TT_SYNC_URL=http://localhost:8080
export TT_SYNC_API_KEY=your-secret-key
```

### Syncing

```bash
tt sync           # Bidirectional sync: push local changes, pull remote changes
tt sync push      # Push only: send local changes to server (no pull)
```

The sync command:
- Pushes all local changes since last sync
- Pulls changes from other clients
- Applies remote changes to your local database

### Resetting Sync

**Reset sync (keep local data, clear server):**

Use this when sync gets into a bad state and you want to start fresh while keeping your local tasks:

```bash
tt sync reset
```

This will:
1. Clear all data on the sync server
2. Clear local sync events and cursor
3. Regenerate sync events from your current local tasks

After reset, run `tt sync` to push your local data to the server.

**Reset client (start fresh locally):**

If you want to reset a client to match the server (discard local changes):

```bash
# Simply delete your local database
rm ~/.local/share/tt/tasks.db    # Default location
# or
rm $TT_DATA_DIR/tasks.db         # If using custom data dir

# Then sync to pull everything from server
tt sync
```

### API Endpoints

The server exposes:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check (no auth required) |
| `/api/v1/sync` | POST | Bidirectional sync |
| `/api/v1/events` | POST | Push events only |
| `/api/v1/sync/reset` | POST | Reset server data |
| `/swagger/` | GET | API documentation |

All `/api/v1/*` endpoints require `Authorization: Bearer <api_key>` header.

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
tt list --sort <TAB>      # Shows sort fields
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
