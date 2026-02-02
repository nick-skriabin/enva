package tui

import (
	"fmt"
	"regexp"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nick-skriabin/enva/internal/shell"
)

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		textinput.Blink,
	)
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.clearToastIfExpired()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Calculate modal content width - use most of screen, max 80
		modalWidth := msg.Width - 20
		if modalWidth > 80 {
			modalWidth = 80
		}
		if modalWidth < 50 {
			modalWidth = 50
		}
		inputWidth := modalWidth - 10 // Account for modal padding + input border/padding
		m.editKeyInput.Width = inputWidth
		m.editValInput.SetWidth(inputWidth)
		m.bulkInput.SetWidth(inputWidth)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Handle text input updates
	var cmd tea.Cmd
	if m.searchFocused && m.modal == ModalNone {
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.searchQuery = m.searchInput.Value()
		m.refreshResults()
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Modal handling takes priority
	if m.modal != ModalNone {
		return m.handleModalKey(msg)
	}

	// Search focused
	if m.searchFocused {
		return m.handleSearchKey(msg)
	}

	// Normal mode
	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "/":
		m.searchFocused = true
		m.searchInput.Focus()
		return m, textinput.Blink

	case "j", "down":
		m.moveDown(1)

	case "k", "up":
		m.moveUp(1)

	case "g":
		m.moveToTop()

	case "G":
		m.moveToBottom()

	case "ctrl+d":
		m.moveDown(m.halfPage())

	case "ctrl+u":
		m.moveUp(m.halfPage())

	case "t":
		// Toggle view mode
		if m.viewMode == ViewEffective {
			m.viewMode = ViewLocal
			m.setToast("Showing local vars only", false)
		} else {
			m.viewMode = ViewEffective
			m.setToast("Showing effective vars", false)
		}
		m.refreshResults()

	case "enter", "e":
		// Edit selected
		if v := m.selectedVar(); v != nil {
			m.openEditModal(v.Key, v.Value, false)
		}

	case "a":
		// Add new
		m.openEditModal("", "", true)

	case "A":
		// Bulk import
		m.openBulkImportModal()

	case "v":
		// View value
		if m.selectedVar() != nil {
			m.modal = ModalView
			m.viewScrollOffset = 0
		}

	case "?":
		// Help
		m.modal = ModalHelp

	case "x":
		// Delete
		if v := m.selectedVar(); v != nil && v.DefinedAtPath == m.ctx.CwdReal {
			m.deleteKey = v.Key
			m.modal = ModalConfirmDelete
		} else if v != nil {
			m.setToast("Can only delete local vars", true)
		}

	case "u":
		// Undo
		return m.handleUndo()

	case "y":
		// Copy KEY=value
		if v := m.selectedVar(); v != nil {
			m.clipboard = fmt.Sprintf("%s=%s", v.Key, v.Value)
			m.setToast("Copied: "+v.Key+"=...", false)
		}

	case "Y":
		// Copy export line
		if v := m.selectedVar(); v != nil {
			m.clipboard = shell.FormatExport(v.Key, v.Value)
			m.setToast("Copied export line", false)
		}

	case "esc":
		if m.searchQuery != "" {
			m.searchQuery = ""
			m.searchInput.SetValue("")
			m.refreshResults()
		}
	}

	return m, nil
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "enter":
		// Exit search focus but keep query
		m.searchFocused = false
		m.searchInput.Blur()
		return m, nil

	case "esc":
		if m.searchQuery != "" {
			// Clear query
			m.searchQuery = ""
			m.searchInput.SetValue("")
			m.refreshResults()
		} else {
			// Exit search focus
			m.searchFocused = false
			m.searchInput.Blur()
		}
		return m, nil

	case "ctrl+c":
		return m, tea.Quit

	case "down":
		m.moveDown(1)
		return m, nil

	case "up":
		m.moveUp(1)
		return m, nil
	}

	// Forward to text input
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.searchQuery = m.searchInput.Value()
	m.refreshResults()
	return m, cmd
}

func (m Model) handleModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch m.modal {
	case ModalEdit:
		return m.handleEditModalKey(msg, key)
	case ModalBulkImport:
		return m.handleBulkImportKey(msg, key)
	case ModalView:
		return m.handleViewModalKey(key)
	case ModalHelp:
		return m.handleHelpModalKey(key)
	case ModalConfirmDelete:
		return m.handleDeleteConfirmKey(key)
	}

	return m, nil
}

func (m Model) handleEditModalKey(msg tea.KeyMsg, key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.modal = ModalNone
		m.editError = ""
		return m, nil

	case "ctrl+s":
		return m.saveEdit()

	case "tab":
		// Switch focus
		if m.editFocus == FocusKey {
			m.editFocus = FocusValue
			m.editKeyInput.Blur()
			m.editValInput.Focus()
		} else {
			m.editFocus = FocusKey
			m.editValInput.Blur()
			m.editKeyInput.Focus()
		}
		return m, nil
	}

	// Forward to focused input
	var cmd tea.Cmd
	if m.editFocus == FocusKey {
		m.editKeyInput, cmd = m.editKeyInput.Update(msg)
	} else {
		m.editValInput, cmd = m.editValInput.Update(msg)
	}
	return m, cmd
}

func (m Model) handleBulkImportKey(msg tea.KeyMsg, key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.modal = ModalNone
		m.bulkError = ""
		return m, nil

	case "ctrl+s":
		return m.saveBulkImport()
	}

	// Forward to textarea
	var cmd tea.Cmd
	m.bulkInput, cmd = m.bulkInput.Update(msg)
	return m, cmd
}

func (m Model) handleViewModalKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q", "v", "enter":
		m.modal = ModalNone
	case "j", "down":
		m.viewScrollOffset++
	case "k", "up":
		if m.viewScrollOffset > 0 {
			m.viewScrollOffset--
		}
	}
	return m, nil
}

func (m Model) handleHelpModalKey(key string) (tea.Model, tea.Cmd) {
	maxLines := m.height - 10
	if maxLines < 5 {
		maxLines = 5
	}
	totalBindings := m.getHelpBindingsCount()
	maxOffset := totalBindings - maxLines
	if maxOffset < 0 {
		maxOffset = 0
	}

	switch key {
	case "esc", "q", "?", "enter":
		m.modal = ModalNone
		m.helpScrollOffset = 0
	case "j", "down":
		if m.helpScrollOffset < maxOffset {
			m.helpScrollOffset++
		}
	case "k", "up":
		if m.helpScrollOffset > 0 {
			m.helpScrollOffset--
		}
	case "g":
		m.helpScrollOffset = 0
	case "G":
		m.helpScrollOffset = maxOffset
	}
	return m, nil
}

func (m Model) handleDeleteConfirmKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y", "Y":
		return m.confirmDelete()
	case "n", "N", "esc":
		m.modal = ModalNone
		m.deleteKey = ""
	}
	return m, nil
}

func (m *Model) openEditModal(key, value string, isNew bool) {
	m.modal = ModalEdit
	m.editIsNew = isNew
	m.editKeyInput.SetValue(key)
	m.editValInput.SetValue(value)
	m.editError = ""

	if isNew {
		m.editFocus = FocusKey
		m.editKeyInput.Focus()
		m.editValInput.Blur()
	} else {
		m.editFocus = FocusValue
		m.editKeyInput.Blur()
		m.editValInput.Focus()
	}
}

func (m *Model) openBulkImportModal() {
	m.modal = ModalBulkImport
	m.bulkInput.SetValue("")
	m.bulkInput.Focus()
	m.bulkError = ""
}

var keyRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func (m Model) saveEdit() (tea.Model, tea.Cmd) {
	key := m.editKeyInput.Value()
	value := m.editValInput.Value()

	// Validate key
	if !keyRegex.MatchString(key) {
		m.editError = "Invalid key: must match [A-Za-z_][A-Za-z0-9_]*"
		return m, nil
	}

	// Save undo info
	oldVar, _ := m.resolver.GetLocalVarsFromDB(m.ctx.CwdReal)
	var hadVal bool
	var oldVal string
	for _, v := range oldVar {
		if v.Key == key {
			hadVal = true
			oldVal = v.Value
			break
		}
	}

	// Set the variable
	if err := m.resolver.SetVar(m.ctx.CwdReal, key, value); err != nil {
		m.editError = fmt.Sprintf("Error: %v", err)
		return m, nil
	}

	// Push undo
	m.pushUndo(UndoAction{
		Type:   "set",
		Key:    key,
		OldVal: oldVal,
		NewVal: value,
		HadVal: hadVal,
	})

	// Reload and close
	if err := m.reloadContext(); err != nil {
		m.setToast(fmt.Sprintf("Reload error: %v", err), true)
	} else {
		if m.editIsNew {
			m.setToast(fmt.Sprintf("Added %s", key), false)
		} else {
			m.setToast(fmt.Sprintf("Updated %s", key), false)
		}
	}

	m.modal = ModalNone
	m.editError = ""
	return m, nil
}

func (m Model) saveBulkImport() (tea.Model, tea.Cmd) {
	content := m.bulkInput.Value()
	parsed, invalid := shell.ParseEnvFile(content)

	if len(invalid) > 0 {
		m.bulkError = fmt.Sprintf("Invalid lines: %v", invalid)
		return m, nil
	}

	if len(parsed) == 0 {
		m.bulkError = "No valid KEY=value lines found"
		return m, nil
	}

	// Get existing for undo
	oldVars, _ := m.resolver.GetLocalVarsFromDB(m.ctx.CwdReal)
	oldMap := make(map[string]string)
	for _, v := range oldVars {
		oldMap[v.Key] = v.Value
	}

	// Set all vars
	if err := m.resolver.SetVarsBatch(m.ctx.CwdReal, parsed); err != nil {
		m.bulkError = fmt.Sprintf("Error: %v", err)
		return m, nil
	}

	// Push undo
	m.pushUndo(UndoAction{
		Type:  "import",
		Batch: oldMap,
	})

	// Reload and close
	added := 0
	updated := 0
	for k := range parsed {
		if _, existed := oldMap[k]; existed {
			updated++
		} else {
			added++
		}
	}

	if err := m.reloadContext(); err != nil {
		m.setToast(fmt.Sprintf("Reload error: %v", err), true)
	} else {
		m.setToast(fmt.Sprintf("Imported %d (added %d, updated %d)", len(parsed), added, updated), false)
	}

	m.modal = ModalNone
	m.bulkError = ""
	return m, nil
}

func (m Model) confirmDelete() (tea.Model, tea.Cmd) {
	key := m.deleteKey

	// Get old value for undo
	var oldVal string
	vars, _ := m.resolver.GetLocalVarsFromDB(m.ctx.CwdReal)
	for _, v := range vars {
		if v.Key == key {
			oldVal = v.Value
			break
		}
	}

	// Delete
	if err := m.resolver.DeleteVar(m.ctx.CwdReal, key); err != nil {
		m.setToast(fmt.Sprintf("Delete error: %v", err), true)
		m.modal = ModalNone
		m.deleteKey = ""
		return m, nil
	}

	// Push undo
	m.pushUndo(UndoAction{
		Type:   "delete",
		Key:    key,
		OldVal: oldVal,
		HadVal: true,
	})

	// Reload
	if err := m.reloadContext(); err != nil {
		m.setToast(fmt.Sprintf("Reload error: %v", err), true)
	} else {
		m.setToast(fmt.Sprintf("Deleted %s", key), false)
	}

	m.modal = ModalNone
	m.deleteKey = ""
	return m, nil
}

func (m Model) handleUndo() (tea.Model, tea.Cmd) {
	action := m.popUndo()
	if action == nil {
		m.setToast("Nothing to undo", true)
		return m, nil
	}

	var err error
	switch action.Type {
	case "set":
		if action.HadVal {
			// Restore old value
			err = m.resolver.SetVar(m.ctx.CwdReal, action.Key, action.OldVal)
		} else {
			// Delete the new key
			err = m.resolver.DeleteVar(m.ctx.CwdReal, action.Key)
		}

	case "delete":
		// Restore deleted key
		err = m.resolver.SetVar(m.ctx.CwdReal, action.Key, action.OldVal)

	case "import":
		// This is complex - we'd need to restore old state
		// For simplicity, just notify user
		m.setToast("Import undo not fully supported", true)
		return m, nil
	}

	if err != nil {
		m.setToast(fmt.Sprintf("Undo error: %v", err), true)
		return m, nil
	}

	if err := m.reloadContext(); err != nil {
		m.setToast(fmt.Sprintf("Reload error: %v", err), true)
	} else {
		m.setToast("Undone", false)
	}

	return m, nil
}
