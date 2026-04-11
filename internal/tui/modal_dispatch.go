package tui

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// modalEntry pairs a modal's active check with closures for update, view, and help keys.
// This allows iterating over all modals generically while keeping type-safe result handling.
type modalEntry struct {
	// active returns whether this modal is currently shown.
	active func() bool
	// update dispatches a message to the modal and handles any result.
	// Returns the updated Model and a tea.Cmd (nil if no action needed).
	update func(m Model, msg tea.Msg) (Model, tea.Cmd)
	// view renders the modal overlay.
	view func() string
	// helpKeys returns the help.KeyMap for the bottom bar while this modal is active.
	helpKeys help.KeyMap
}

// modalEntries returns all modals as a slice of modalEntry.
// The order determines priority: the first active modal wins.
func (m *Model) modalEntries() []modalEntry {
	return []modalEntry{
		{
			active: m.addModal.Active,
			update: func(mdl Model, msg tea.Msg) (Model, tea.Cmd) {
				var result *AddResult
				mdl.addModal, result = mdl.addModal.Update(msg)
				if result != nil && !result.Canceled {
					return mdl, mdl.createTask(result)
				}
				return mdl, nil
			},
			view:     m.addModal.View,
			helpKeys: addKeys,
		},
		{
			active: m.renameModal.Active,
			update: func(mdl Model, msg tea.Msg) (Model, tea.Cmd) {
				var result *RenameResult
				mdl.renameModal, result = mdl.renameModal.Update(msg)
				if result != nil && !result.Canceled {
					if result.ItemType == "area" {
						return mdl, mdl.renameArea(result.ItemKey, result.NewTitle)
					}
					return mdl, mdl.renameTask(result.TaskID, result.NewTitle)
				}
				return mdl, nil
			},
			view:     m.renameModal.View,
			helpKeys: renameKeys,
		},
		{
			active: m.moveModal.Active,
			update: func(mdl Model, msg tea.Msg) (Model, tea.Cmd) {
				var result *MoveResult
				mdl.moveModal, result = mdl.moveModal.Update(msg)
				if result != nil && !result.Canceled {
					return mdl, mdl.moveTask(result.TaskID, result.ItemType, result.Name)
				}
				return mdl, nil
			},
			view:     m.moveModal.View,
			helpKeys: moveKeys,
		},
		{
			active: m.dateModal.Active,
			update: func(mdl Model, msg tea.Msg) (Model, tea.Cmd) {
				var result *DateResult
				mdl.dateModal, result = mdl.dateModal.Update(msg)
				if result != nil && !result.Canceled {
					return mdl, mdl.setTaskDate(result.TaskID, result.Date, result.Mode)
				}
				return mdl, nil
			},
			view: m.dateModal.View,
			helpKeys: func() help.KeyMap {
				if m.dateModal.FocusInput() {
					return dateInputKeys
				}
				return datePickerKeys
			}(),
		},
		{
			active: m.tagModal.Active,
			update: func(mdl Model, msg tea.Msg) (Model, tea.Cmd) {
				var result *TagResult
				mdl.tagModal, result = mdl.tagModal.Update(msg)
				if result != nil && !result.Canceled {
					return mdl, mdl.setTaskTags(result.TaskID, result.Tags)
				}
				return mdl, nil
			},
			view:     m.tagModal.View,
			helpKeys: tagKeys,
		},
		{
			active: m.descriptionModal.Active,
			update: func(mdl Model, msg tea.Msg) (Model, tea.Cmd) {
				var result *DescriptionResult
				mdl.descriptionModal, result = mdl.descriptionModal.Update(msg)
				if result != nil && !result.Canceled {
					return mdl, mdl.setTaskDescription(result.TaskID, result.Description)
				}
				return mdl, nil
			},
			view:     m.descriptionModal.View,
			helpKeys: descriptionKeys,
		},
		{
			active: m.confirmModal.Active,
			update: func(mdl Model, msg tea.Msg) (Model, tea.Cmd) {
				var result *ConfirmResult
				mdl.confirmModal, result = mdl.confirmModal.Update(msg)
				if result != nil && result.Confirmed {
					return mdl, mdl.deleteItem(result)
				}
				return mdl, nil
			},
			view:     m.confirmModal.View,
			helpKeys: confirmKeys,
		},
		{
			active: m.completeModal.Active,
			update: func(mdl Model, msg tea.Msg) (Model, tea.Cmd) {
				var result *CompleteResult
				mdl.completeModal, result = mdl.completeModal.Update(msg)
				if result != nil && result.Confirmed {
					return mdl, mdl.completeProject(result)
				}
				return mdl, nil
			},
			view:     m.completeModal.View,
			helpKeys: confirmKeys, // completeModal uses same keys as confirm
		},
		{
			active: m.createProjectModal.Active,
			update: func(mdl Model, msg tea.Msg) (Model, tea.Cmd) {
				var result *CreateProjectResult
				mdl.createProjectModal, result = mdl.createProjectModal.Update(msg)
				if result != nil && !result.Canceled {
					return mdl, mdl.createProject(result)
				}
				return mdl, nil
			},
			view:     m.createProjectModal.View,
			helpKeys: createProjectKeys,
		},
		{
			active: m.createAreaModal.Active,
			update: func(mdl Model, msg tea.Msg) (Model, tea.Cmd) {
				var result *CreateAreaResult
				mdl.createAreaModal, result = mdl.createAreaModal.Update(msg)
				if result != nil && !result.Canceled {
					return mdl, mdl.createArea(result)
				}
				return mdl, nil
			},
			view:     m.createAreaModal.View,
			helpKeys: createAreaKeys,
		},
		{
			active: m.createNoteModal.Active,
			update: func(mdl Model, msg tea.Msg) (Model, tea.Cmd) {
				var result *CreateNoteResult
				mdl.createNoteModal, result = mdl.createNoteModal.Update(msg)
				if result != nil && !result.Canceled {
					return mdl, mdl.createAndOpenNote(result)
				}
				return mdl, nil
			},
			view:     m.createNoteModal.View,
			helpKeys: createNoteKeys,
		},
		{
			active: m.helpModal.Active,
			update: func(mdl Model, msg tea.Msg) (Model, tea.Cmd) {
				var closed bool
				mdl.helpModal, closed = mdl.helpModal.Update(msg)
				_ = closed
				return mdl, nil
			},
			view:     m.helpModal.View,
			helpKeys: helpModalKeys,
		},
	}
}

// updateActiveModal dispatches a message to the first active modal.
// Returns true if a modal was active and handled the message.
func (m Model) updateActiveModal(msg tea.Msg) (Model, tea.Cmd, bool) {
	entries := m.modalEntries()
	for _, entry := range entries {
		if entry.active() {
			mdl, cmd := entry.update(m, msg)
			return mdl, cmd, true
		}
	}
	return m, nil, false
}

// activeModalView returns the view string for the first active modal,
// along with the appropriate help bar. Returns empty string if no modal is active.
func (m Model) activeModalView() (modalView string, helpKeyMap help.KeyMap, found bool) {
	entries := m.modalEntries()
	for _, entry := range entries {
		if entry.active() {
			return entry.view(), entry.helpKeys, true
		}
	}
	return "", nil, false
}

// renderModalOverlay renders a modal with the help bar, or returns empty if no modal is active.
func (m Model) renderModalOverlay() (string, bool) {
	modalView, helpKeyMap, found := m.activeModalView()
	if !found {
		return "", false
	}
	helpView := m.help.View(helpKeyMap)
	helpView = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, helpView)
	return lipgloss.JoinVertical(lipgloss.Left, modalView, helpView), true
}
