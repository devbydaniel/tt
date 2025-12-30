package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up           key.Binding
	Down         key.Binding
	Tab          key.Binding
	ShiftTab     key.Binding
	Enter        key.Binding
	Escape       key.Binding
	FocusSidebar key.Binding
	FocusContent key.Binding
	Rename       key.Binding
	Move         key.Binding
	Planned      key.Binding
	Due          key.Binding
	Add          key.Binding
	Quit         key.Binding
}

// sidebarKeyMap provides help bindings when sidebar is focused
type sidebarKeyMap struct{}

func (k sidebarKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{keys.Up, keys.Down, keys.Tab, keys.FocusContent, keys.Add, keys.Quit}
}

func (k sidebarKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{keys.Up, keys.Down, keys.Tab, keys.ShiftTab, keys.FocusContent, keys.Quit}}
}

// contentKeyMap provides help bindings when content is focused
type contentKeyMap struct{}

func (k contentKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{keys.Up, keys.Down, keys.FocusSidebar, keys.Rename, keys.Move, keys.Planned, keys.Due, keys.Add, keys.Quit}
}

func (k contentKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{keys.Up, keys.Down, keys.FocusSidebar, keys.Rename, keys.Move, keys.Planned, keys.Due, keys.Quit}}
}

// renameKeyMap provides help bindings for rename modal
type renameKeyMap struct{}

func (k renameKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{keys.Enter, keys.Escape}
}

func (k renameKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{keys.Enter, keys.Escape}}
}

// moveKeyMap provides help bindings for move modal
type moveKeyMap struct{}

func (k moveKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{keys.Up, keys.Down, keys.Enter, keys.Escape}
}

func (k moveKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{keys.Up, keys.Down, keys.Enter, keys.Escape}}
}

// dateInputKeyMap provides help bindings for date modal when input is focused
type dateInputKeyMap struct{}

func (k dateInputKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "date picker")),
		keys.Enter,
		keys.Escape,
	}
}

func (k dateInputKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

// datePickerKeyMap provides help bindings for date modal when picker is focused
type datePickerKeyMap struct{}

func (k datePickerKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "input")),
		key.NewBinding(key.WithKeys("left", "up", "right", "down"), key.WithHelp("←↑→↓", "navigate")),
		keys.Enter,
		keys.Escape,
	}
}

func (k datePickerKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

// addKeyMap provides help bindings for add modal
type addKeyMap struct{}

func (k addKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
		key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("s-tab", "prev field")),
		keys.Enter,
		keys.Escape,
	}
}

func (k addKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

var (
	sidebarKeys    = sidebarKeyMap{}
	contentKeys    = contentKeyMap{}
	renameKeys     = renameKeyMap{}
	moveKeys       = moveKeyMap{}
	dateInputKeys  = dateInputKeyMap{}
	datePickerKeys = datePickerKeyMap{}
	addKeys        = addKeyMap{}
)

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k/up", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j/down", "move down"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next section"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev section"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	FocusSidebar: key.NewBinding(
		key.WithKeys("h"),
		key.WithHelp("h", "sidebar"),
	),
	FocusContent: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "content"),
	),
	Rename: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "rename"),
	),
	Move: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "move"),
	),
	Planned: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "set planned date"),
	),
	Due: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "set due date"),
	),
	Add: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add task"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
