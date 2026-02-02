package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"enva/internal/env"
	"enva/internal/search"
)

// ensure import is used
var _ = search.SearchResult{}

// View renders the UI.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Handle modals
	switch m.modal {
	case ModalEdit:
		return m.renderEditModal()
	case ModalBulkImport:
		return m.renderBulkImportModal()
	case ModalView:
		return m.renderViewModal()
	case ModalHelp:
		return m.renderHelpModal()
	case ModalConfirmDelete:
		return m.renderDeleteConfirmModal()
	}

	var b strings.Builder

	// Top bar row 1: Root and Profile
	topBar1 := m.renderTopBar1()
	b.WriteString(topBar1)
	b.WriteString("\n")

	// Top bar row 2: Search
	topBar2 := m.renderTopBar2()
	b.WriteString(topBar2)
	b.WriteString("\n")

	// Table
	table := m.renderTable()
	b.WriteString(table)

	// Status bar
	statusBar := m.renderStatusBar()
	b.WriteString(statusBar)

	return b.String()
}

func (m Model) renderTopBar1() string {
	rootLabel := styleRoot.Render("Root: " + m.ctx.RootDir)
	profileLabel := styleProfile.Render("Profile: " + m.ctx.Profile)

	// View mode indicator
	viewModeStr := "Effective"
	if m.viewMode == ViewLocal {
		viewModeStr = "Local"
	}
	viewLabel := styleProfile.Render("[" + viewModeStr + "]")

	// Pad to fill width
	left := rootLabel
	right := viewLabel + "  " + profileLabel
	padding := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if padding < 1 {
		padding = 1
	}

	content := left + strings.Repeat(" ", padding) + right
	return styleTopBar.Width(m.width).Render(content)
}

func (m Model) renderTopBar2() string {
	searchLabel := styleSearchLabel.Render("Search: ")
	var searchContent string
	if m.searchFocused {
		searchContent = m.searchInput.View()
	} else if m.searchQuery != "" {
		searchContent = styleSearchQuery.Render(m.searchQuery)
	} else {
		searchContent = styleSecondary.Render("(press / to search)")
	}

	return styleTopBar.Width(m.width).Render(searchLabel + searchContent)
}

var styleSecondary = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

func (m Model) renderTable() string {
	visibleRows := m.visibleRows()
	tableHeight := visibleRows + 1 // +1 for header

	// Calculate column widths
	keyColWidth := 30
	valueColWidth := m.width - keyColWidth - 6 // 6 for badge, borders, padding
	if valueColWidth < 20 {
		valueColWidth = 20
	}

	var lines []string

	// Header
	header := fmt.Sprintf("  %-*s  %-*s", keyColWidth, "Key", valueColWidth, "Value")
	lines = append(lines, styleTableHeader.Width(m.width).Render(header))

	// Rows
	endIdx := m.offset + visibleRows
	if endIdx > len(m.results) {
		endIdx = len(m.results)
	}

	for i := m.offset; i < endIdx; i++ {
		result := m.results[i]
		v := result.Var
		isSelected := i == m.cursor

		// Badge
		badge := m.getBadge(v)

		// Key - truncate and pad to fixed width, then highlight
		keyStr := truncate(v.Key, keyColWidth)
		keyPadded := fmt.Sprintf("%-*s", keyColWidth, keyStr)
		if m.searchQuery != "" && len(result.KeyMatches) > 0 {
			keyPadded = highlightMatchesPadded(keyStr, keyColWidth, result.KeyMatches)
		}

		// Value - truncate and pad to fixed width, then highlight
		valueStr := truncate(singleLine(v.Value), valueColWidth)
		valuePadded := fmt.Sprintf("%-*s", valueColWidth, valueStr)
		if m.searchQuery != "" && len(result.ValueMatches) > 0 {
			valuePadded = highlightMatchesPadded(valueStr, valueColWidth, result.ValueMatches)
		}

		row := fmt.Sprintf("%s %s  %s", badge, keyPadded, valuePadded)

		if isSelected {
			lines = append(lines, styleTableRowSelected.Width(m.width).Render(row))
		} else {
			lines = append(lines, styleTableRow.Width(m.width).Render(row))
		}
	}

	// Pad remaining rows
	for len(lines) < tableHeight {
		lines = append(lines, strings.Repeat(" ", m.width))
	}

	return strings.Join(lines, "\n") + "\n"
}

func (m Model) getBadge(v *env.ResolvedVar) string {
	if v.DefinedAtPath == m.ctx.CwdReal {
		if v.Overrode {
			return styleBadgeOverride.Render(badgeOverride)
		}
		return styleBadgeLocal.Render(badgeLocal)
	}
	return styleBadgeInherited.Render(badgeInherited)
}

func (m Model) renderStatusBar() string {
	var left, right string

	// Selected var info
	if v := m.selectedVar(); v != nil {
		left = styleStatusKey.Render("Defined at: ") + styleStatusValue.Render(v.DefinedAtPath)
		if v.Overrode {
			left += "  " + styleStatusKey.Render("Overrides: ") + styleStatusValue.Render(v.OverrodePath)
		}
	}

	// Toast message
	if m.toast != "" {
		if m.toastIsErr {
			right = styleToastError.Render(m.toast)
		} else {
			right = styleToast.Render(m.toast)
		}
	}

	// Count
	countStr := fmt.Sprintf("%d/%d", m.cursor+1, len(m.results))
	right = countStr + "  " + right

	padding := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if padding < 1 {
		padding = 1
	}

	content := left + strings.Repeat(" ", padding) + right
	return styleStatusBar.Width(m.width).Render(content)
}

func (m Model) renderEditModal() string {
	title := "Edit Variable"
	if m.editIsNew {
		title = "Add Variable"
	}

	// Modal width - use most of screen width, max 80
	modalWidth := m.width - 20
	if modalWidth > 80 {
		modalWidth = 80
	}
	if modalWidth < 50 {
		modalWidth = 50
	}
	inputWidth := modalWidth - 6 // Account for modal padding + input border

	var content strings.Builder
	content.WriteString(styleModalTitle.Render(title))
	content.WriteString("\n")

	// Key field
	content.WriteString(styleModalLabel.Render("Key:"))
	content.WriteString("\n")
	keyInput := m.editKeyInput.View()
	if m.editFocus == FocusKey {
		content.WriteString(styleModalInputFocused.Width(inputWidth).Render(keyInput))
	} else {
		content.WriteString(styleModalInput.Width(inputWidth).Render(keyInput))
	}
	content.WriteString("\n")

	// Value field
	content.WriteString(styleModalLabel.Render("Value:"))
	content.WriteString("\n")
	valInput := m.editValInput.View()
	if m.editFocus == FocusValue {
		content.WriteString(styleModalInputFocused.Width(inputWidth).Render(valInput))
	} else {
		content.WriteString(styleModalInput.Width(inputWidth).Render(valInput))
	}

	// Error
	if m.editError != "" {
		content.WriteString("\n")
		content.WriteString(styleError.Render(m.editError))
	}

	// Help
	content.WriteString("\n")
	content.WriteString(styleHelpDesc.Render("Tab: switch field  Ctrl+S: save  Esc: cancel"))

	modal := styleModalBox.Width(modalWidth).Render(content.String())
	return centerModal(modal, m.width, m.height)
}

func (m Model) renderBulkImportModal() string {
	// Modal width - use most of screen, max 80
	modalWidth := m.width - 20
	if modalWidth > 80 {
		modalWidth = 80
	}
	if modalWidth < 50 {
		modalWidth = 50
	}
	inputWidth := modalWidth - 6

	var content strings.Builder
	content.WriteString(styleModalTitle.Render("Bulk Import"))
	content.WriteString("\n")

	content.WriteString(styleModalLabel.Render("Enter KEY=value lines:"))
	content.WriteString("\n")
	content.WriteString(styleModalInputFocused.Width(inputWidth).Render(m.bulkInput.View()))

	// Error
	if m.bulkError != "" {
		content.WriteString("\n")
		content.WriteString(styleError.Render(m.bulkError))
	}

	// Help
	content.WriteString("\n")
	content.WriteString(styleHelpDesc.Render("Formats: KEY=value, export KEY=value, # comments"))
	content.WriteString("\n")
	content.WriteString(styleHelpDesc.Render("Ctrl+S: import  Esc: cancel"))

	modal := styleModalBox.Width(modalWidth).Render(content.String())
	return centerModal(modal, m.width, m.height)
}

func (m Model) renderViewModal() string {
	v := m.selectedVar()
	if v == nil {
		return centerModal(styleModalBox.Render("No variable selected"), m.width, m.height)
	}

	var content strings.Builder
	content.WriteString(styleModalTitle.Render("Value: " + v.Key))
	content.WriteString("\n\n")

	// Show value with scroll
	lines := strings.Split(v.Value, "\n")
	maxLines := m.height - 10
	if maxLines < 5 {
		maxLines = 5
	}

	startLine := m.viewScrollOffset
	if startLine > len(lines)-1 {
		startLine = len(lines) - 1
	}
	if startLine < 0 {
		startLine = 0
	}

	endLine := startLine + maxLines
	if endLine > len(lines) {
		endLine = len(lines)
	}

	for i := startLine; i < endLine; i++ {
		content.WriteString(lines[i])
		if i < endLine-1 {
			content.WriteString("\n")
		}
	}

	if len(lines) > maxLines {
		content.WriteString("\n\n")
		content.WriteString(styleHelpDesc.Render(fmt.Sprintf("Lines %d-%d of %d (j/k to scroll)", startLine+1, endLine, len(lines))))
	}

	content.WriteString("\n\n")
	content.WriteString(styleHelpDesc.Render("Esc/q/v: close"))

	modal := styleModalBox.Width(m.width - 4).Render(content.String())
	return centerModal(modal, m.width, m.height)
}

func (m Model) renderHelpModal() string {
	var content strings.Builder
	content.WriteString(styleModalTitle.Render("Keybindings"))
	content.WriteString("\n\n")

	bindings := []struct{ key, desc string }{
		{"j/k, ↑/↓", "Navigate up/down"},
		{"g/G", "Go to top/bottom"},
		{"Ctrl+d/u", "Half page down/up"},
		{"/", "Enter search mode"},
		{"Esc", "Clear search / exit search"},
		{"t", "Toggle view: Effective / Local"},
		{"Enter, e", "Edit selected variable"},
		{"a", "Add new variable"},
		{"A", "Bulk import variables"},
		{"v", "View full value"},
		{"x", "Delete local variable"},
		{"u", "Undo last action"},
		{"y", "Copy KEY=value"},
		{"Y", "Copy export line"},
		{"?", "Show this help"},
		{"q", "Quit"},
	}

	for _, b := range bindings {
		content.WriteString(styleHelpKey.Render(fmt.Sprintf("%-12s", b.key)))
		content.WriteString(styleHelpDesc.Render(b.desc))
		content.WriteString("\n")
	}

	content.WriteString("\n")
	content.WriteString(styleHelpDesc.Render("Press Esc or ? to close"))

	modal := styleModalBox.Render(content.String())
	return centerModal(modal, m.width, m.height)
}

func (m Model) renderDeleteConfirmModal() string {
	var content strings.Builder
	content.WriteString(styleConfirm.Render(fmt.Sprintf("Delete %s?", m.deleteKey)))
	content.WriteString("\n\n")
	content.WriteString(styleHelpDesc.Render("y: confirm  n/Esc: cancel"))

	modal := styleModalBox.Render(content.String())
	return centerModal(modal, m.width, m.height)
}

// Helper functions

func centerModal(modal string, width, height int) string {
	modalWidth := lipgloss.Width(modal)
	modalHeight := lipgloss.Height(modal)

	padLeft := (width - modalWidth) / 2
	padTop := (height - modalHeight) / 2

	if padLeft < 0 {
		padLeft = 0
	}
	if padTop < 0 {
		padTop = 0
	}

	var result strings.Builder
	for i := 0; i < padTop; i++ {
		result.WriteString(strings.Repeat(" ", width))
		result.WriteString("\n")
	}

	lines := strings.Split(modal, "\n")
	for _, line := range lines {
		result.WriteString(strings.Repeat(" ", padLeft))
		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func singleLine(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

func highlightMatches(s string, indices []int) string {
	if len(indices) == 0 {
		return s
	}

	indexSet := make(map[int]bool)
	for _, i := range indices {
		indexSet[i] = true
	}

	highlighted := styleMatchHighlight
	normal := lipgloss.NewStyle()

	var result strings.Builder
	inHighlight := false

	runes := []rune(s)
	for i, r := range runes {
		shouldHighlight := indexSet[i]
		if shouldHighlight && !inHighlight {
			inHighlight = true
		} else if !shouldHighlight && inHighlight {
			inHighlight = false
		}

		if inHighlight {
			result.WriteString(highlighted.Render(string(r)))
		} else {
			result.WriteString(normal.Render(string(r)))
		}
	}

	return result.String()
}

// highlightMatchesPadded highlights matches and pads to width (accounting for ANSI codes)
func highlightMatchesPadded(s string, width int, indices []int) string {
	indexSet := make(map[int]bool)
	for _, i := range indices {
		indexSet[i] = true
	}

	highlighted := styleMatchHighlight
	normal := lipgloss.NewStyle()

	var result strings.Builder
	runes := []rune(s)
	visualLen := len(runes)

	for i, r := range runes {
		if indexSet[i] {
			result.WriteString(highlighted.Render(string(r)))
		} else {
			result.WriteString(normal.Render(string(r)))
		}
	}

	// Pad with spaces to reach desired width
	if visualLen < width {
		result.WriteString(strings.Repeat(" ", width-visualLen))
	}

	return result.String()
}
