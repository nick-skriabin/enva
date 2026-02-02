package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"

	"github.com/nick-skriabin/enva/internal/db"
	"github.com/nick-skriabin/enva/internal/env"
	"github.com/nick-skriabin/enva/internal/search"
)

// ViewMode represents the current list view mode.
type ViewMode int

const (
	ViewEffective ViewMode = iota // Show merged effective vars
	ViewLocal                     // Show only local vars
)

// ModalType represents the type of modal currently displayed.
type ModalType int

const (
	ModalNone          ModalType = iota
	ModalEdit                    // Edit/Add variable
	ModalBulkImport              // Bulk import
	ModalView                    // Read-only value view
	ModalHelp                    // Help/keybindings
	ModalConfirmDelete           // Delete confirmation
)

// FocusField represents which field is focused in edit modal.
type FocusField int

const (
	FocusKey FocusField = iota
	FocusValue
)

// UndoAction represents an action that can be undone.
type UndoAction struct {
	Type    string // "set", "delete", "import"
	Key     string
	OldVal  string            // Previous value (for set/delete)
	NewVal  string            // New value (for set)
	HadVal  bool              // Whether there was a previous value
	Batch   map[string]string // For import undo
	Deleted []string          // Keys that were deleted in the batch
}

// Model is the main TUI model.
type Model struct {
	// Data
	db       *db.DB
	resolver *env.Resolver
	ctx      *env.ResolveContext

	// UI state
	width         int
	height        int
	cursor        int // Selected row index
	offset        int // Scroll offset
	viewMode      ViewMode
	searchFocused bool
	searchQuery   string

	// Search input
	searchInput textinput.Model

	// Filtered/searched results
	results []*search.SearchResult

	// Modal state
	modal        ModalType
	editIsNew    bool // true if adding new var
	editKeyInput textinput.Model
	editValInput textarea.Model
	editFocus    FocusField
	editError    string

	// Bulk import
	bulkInput textarea.Model
	bulkError string

	// View modal
	viewScrollOffset int

	// Help modal
	helpScrollOffset int

	// Delete confirmation
	deleteKey string

	// Toast/status message
	toast       string
	toastExpiry time.Time
	toastIsErr  bool

	// Undo
	undoStack []UndoAction

	// For clipboard (optional feature)
	clipboard string
}

// NewModel creates a new TUI model.
func NewModel(database *db.DB, resolver *env.Resolver, ctx *env.ResolveContext) Model {
	// Search input
	si := textinput.New()
	si.Placeholder = "Type to search..."
	si.CharLimit = 100

	// Edit key input
	ki := textinput.New()
	ki.Placeholder = "KEY_NAME"
	ki.CharLimit = 256

	// Edit value textarea
	vi := textarea.New()
	vi.Placeholder = "value"
	vi.CharLimit = 65536
	vi.SetHeight(5)

	// Bulk import textarea
	bi := textarea.New()
	bi.Placeholder = "KEY=value\nexport KEY2=value2\n# comment"
	bi.CharLimit = 1000000
	bi.SetHeight(15)

	m := Model{
		db:           database,
		resolver:     resolver,
		ctx:          ctx,
		viewMode:     ViewEffective,
		searchInput:  si,
		editKeyInput: ki,
		editValInput: vi,
		bulkInput:    bi,
		undoStack:    make([]UndoAction, 0),
	}

	m.refreshResults()
	return m
}

// refreshResults updates the search results based on current view and query.
func (m *Model) refreshResults() {
	var vars []*env.ResolvedVar

	switch m.viewMode {
	case ViewEffective:
		vars = m.ctx.GetSortedVars()
	case ViewLocal:
		vars = m.ctx.GetLocalVars()
	}

	m.results = search.Search(vars, m.searchQuery)

	// Ensure cursor is within bounds
	if m.cursor >= len(m.results) {
		m.cursor = len(m.results) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// reloadContext reloads the environment context from the database.
func (m *Model) reloadContext() error {
	newCtx, err := m.resolver.Resolve(m.ctx.CwdReal)
	if err != nil {
		return err
	}
	m.ctx = newCtx
	m.refreshResults()
	return nil
}

// selectedVar returns the currently selected variable, or nil if none.
func (m *Model) selectedVar() *env.ResolvedVar {
	if m.cursor >= 0 && m.cursor < len(m.results) {
		return m.results[m.cursor].Var
	}
	return nil
}

// selectedResult returns the currently selected search result, or nil if none.
func (m *Model) selectedResult() *search.SearchResult {
	if m.cursor >= 0 && m.cursor < len(m.results) {
		return m.results[m.cursor]
	}
	return nil
}

// isSelectedLocal returns true if the selected var is local.
func (m *Model) isSelectedLocal() bool {
	v := m.selectedVar()
	return v != nil && v.DefinedAtPath == m.ctx.CwdReal
}

// setToast sets a toast message.
func (m *Model) setToast(msg string, isErr bool) {
	m.toast = msg
	m.toastIsErr = isErr
	m.toastExpiry = time.Now().Add(3 * time.Second)
}

// clearToastIfExpired clears the toast if it has expired.
func (m *Model) clearToastIfExpired() {
	if m.toast != "" && time.Now().After(m.toastExpiry) {
		m.toast = ""
	}
}

// pushUndo pushes an undo action onto the stack (max 1 for simplicity).
func (m *Model) pushUndo(action UndoAction) {
	m.undoStack = []UndoAction{action} // Only keep last action
}

// popUndo pops and returns the last undo action, or nil if empty.
func (m *Model) popUndo() *UndoAction {
	if len(m.undoStack) == 0 {
		return nil
	}
	action := m.undoStack[len(m.undoStack)-1]
	m.undoStack = m.undoStack[:len(m.undoStack)-1]
	return &action
}

// visibleRows returns the number of visible table rows.
func (m *Model) visibleRows() int {
	// Height minus: top bar (1), border (2), header+separator (2), help bar (1)
	rows := m.height - 6
	if rows < 1 {
		rows = 1
	}
	return rows
}

// ensureCursorVisible adjusts offset to keep cursor visible.
func (m *Model) ensureCursorVisible() {
	visible := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
}

// moveUp moves the cursor up.
func (m *Model) moveUp(n int) {
	m.cursor -= n
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.ensureCursorVisible()
}

// moveDown moves the cursor down.
func (m *Model) moveDown(n int) {
	m.cursor += n
	if m.cursor >= len(m.results) {
		m.cursor = len(m.results) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.ensureCursorVisible()
}

// moveToTop moves cursor to first item.
func (m *Model) moveToTop() {
	m.cursor = 0
	m.offset = 0
}

// moveToBottom moves cursor to last item.
func (m *Model) moveToBottom() {
	m.cursor = len(m.results) - 1
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.ensureCursorVisible()
}

// halfPage returns half the visible rows.
func (m *Model) halfPage() int {
	hp := m.visibleRows() / 2
	if hp < 1 {
		hp = 1
	}
	return hp
}
