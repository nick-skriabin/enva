package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/nick-skriabin/enva/internal/env"
	"github.com/nick-skriabin/enva/internal/search"
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

	// Top bar: enva │ Search: query                    profile
	b.WriteString(m.renderTopBar())
	b.WriteString("\n")

	// Main content with border
	b.WriteString(m.renderMainContent())

	// Bottom help bar
	b.WriteString(m.renderHelpBar())

	return b.String()
}

func (m Model) renderTopBar() string {
	// Left side: app name and search
	appName := styleAppName.Render("enva")
	sep := styleDim.Render(" │ ")

	var searchPart string
	if m.searchFocused {
		searchPart = styleDim.Render("Search: ") + m.searchInput.View()
	} else if m.searchQuery != "" {
		searchPart = styleDim.Render("Search: ") + styleSearchQuery.Render(m.searchQuery)
	} else {
		searchPart = styleDim.Render("Search: ") + styleDim.Render("...")
	}

	left := appName + sep + searchPart

	// Right side: profile
	right := styleDim.Render(m.ctx.Profile)

	padding := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if padding < 1 {
		padding = 1
	}

	return left + strings.Repeat(" ", padding) + right
}

func (m Model) renderMainContent() string {
	// Calculate available height for table (total - top bar - help bar - horizontal lines)
	contentHeight := m.height - 4
	if contentHeight < 3 {
		contentHeight = 3
	}

	// Title line
	viewMode := "Effective"
	if m.viewMode == ViewLocal {
		viewMode = "Local"
	}
	title := fmt.Sprintf("%s Variables (%d/%d)", viewMode, m.cursor+1, len(m.results))

	var b strings.Builder

	// Top horizontal line with title
	titleStyled := styleBorderTitle.Render(title)
	lineWidth := m.width - lipgloss.Width(titleStyled) - 3
	if lineWidth < 0 {
		lineWidth = 0
	}
	b.WriteString(styleDim.Render("─ "))
	b.WriteString(titleStyled)
	b.WriteString(styleDim.Render(" " + strings.Repeat("─", lineWidth)))
	b.WriteString("\n")

	// Table content
	b.WriteString(m.renderTableContent(contentHeight))

	// Bottom horizontal line
	b.WriteString("\n")
	b.WriteString(styleDim.Render(strings.Repeat("─", m.width)))
	b.WriteString("\n")

	return b.String()
}

func (m Model) renderTableContent(height int) string {
	// Column widths - border takes 1 char each side
	innerWidth := m.width - 4
	keyColWidth := 24
	sourceColWidth := 10
	descColWidth := 20
	// Row format: " key  value  desc  source"
	// Widths: 1 + key + 2 + value + 2 + desc + 2 + source
	valueColWidth := innerWidth - keyColWidth - descColWidth - sourceColWidth - 7
	if valueColWidth < 15 {
		valueColWidth = 15
	}

	var lines []string

	// Header
	header := fmt.Sprintf(" %-*s  %-*s  %-*s  %-*s",
		keyColWidth, "Key",
		valueColWidth, "Value",
		descColWidth, "Description",
		sourceColWidth, "Source")
	lines = append(lines, styleTableHeader.Render(header))

	// Separator - horizontal line
	sepLine := strings.Repeat("─", innerWidth)
	lines = append(lines, styleDim.Render(sepLine))

	// Data rows
	visibleRows := height - 2 // minus header and separator
	if visibleRows < 1 {
		visibleRows = 1
	}

	endIdx := m.offset + visibleRows
	if endIdx > len(m.results) {
		endIdx = len(m.results)
	}

	for i := m.offset; i < endIdx; i++ {
		result := m.results[i]
		v := result.Var
		isSelected := i == m.cursor

		// Key
		keyStr := fmt.Sprintf("%-*s", keyColWidth, truncate(v.Key, keyColWidth))

		// Value
		valueStr := fmt.Sprintf("%-*s", valueColWidth, truncate(singleLine(v.Value), valueColWidth))

		// Description
		descStr := fmt.Sprintf("%-*s", descColWidth, truncate(v.Description, descColWidth))

		// Source
		sourceStr := fmt.Sprintf("%-*s", sourceColWidth, m.getSourceText(v))

		if isSelected {
			// Build plain row and apply selection style
			row := fmt.Sprintf(" %s  %s  %s  %s", keyStr, valueStr, descStr, sourceStr)
			row = padToWidth(row, innerWidth)
			lines = append(lines, styleTableRowSelected.Render(row))
		} else {
			// Apply search highlighting and source coloring
			if m.searchQuery != "" && len(result.KeyMatches) > 0 {
				keyStr = highlightMatchesPadded(truncate(v.Key, keyColWidth), keyColWidth, result.KeyMatches)
			}
			if m.searchQuery != "" && len(result.ValueMatches) > 0 {
				valueStr = highlightMatchesPadded(truncate(singleLine(v.Value), valueColWidth), valueColWidth, result.ValueMatches)
			}
			// Description in dim style when not selected
			descStyled := styleDim.Render(descStr)
			sourceStyled := m.getSourceBadge(v)

			row := " " + keyStr + "  " + valueStr + "  " + descStyled + "  " + sourceStyled
			lines = append(lines, row)
		}
	}

	// Pad remaining rows
	for len(lines) < height {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func (m Model) getSourceText(v *env.ResolvedVar) string {
	if v.DefinedAtPath == m.ctx.CwdReal {
		if v.Overrode {
			return "Override"
		}
		return "Local"
	}
	return "Inherited"
}

func (m Model) getSourceBadge(v *env.ResolvedVar) string {
	width := 10
	if v.DefinedAtPath == m.ctx.CwdReal {
		if v.Overrode {
			return styleBadgeOverride.Render(fmt.Sprintf("%-*s", width, "Override"))
		}
		return styleBadgeLocal.Render(fmt.Sprintf("%-*s", width, "Local"))
	}
	return styleBadgeInherited.Render(fmt.Sprintf("%-*s", width, "Inherited"))
}

func (m Model) renderHelpBar() string {
	// Keybindings help
	help := []struct{ key, desc string }{
		{"Esc", "Quit"},
		{"e", "Edit"},
		{"a", "Add"},
		{"x", "Delete"},
		{"?", "Help"},
	}

	var parts []string
	for _, h := range help {
		parts = append(parts, styleHelpKey.Render(h.key)+" "+styleDim.Render(h.desc))
	}
	left := strings.Join(parts, "  ")

	// Toast or position
	var right string
	if m.toast != "" {
		if m.toastIsErr {
			right = styleToastError.Render(m.toast)
		} else {
			right = styleToast.Render(m.toast)
		}
	} else {
		right = styleDim.Render(fmt.Sprintf("Item %d of %d", m.cursor+1, len(m.results)))
	}

	padding := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if padding < 1 {
		padding = 1
	}

	return left + strings.Repeat(" ", padding) + right
}

func (m Model) renderStatusBar() string {
	// Kept for compatibility but not used in new design
	return ""
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
	content.WriteString("\n")

	// Description field
	content.WriteString(styleModalLabel.Render("Description (optional):"))
	content.WriteString("\n")
	descInput := m.editDescInput.View()
	if m.editFocus == FocusDescription {
		content.WriteString(styleModalInputFocused.Width(inputWidth).Render(descInput))
	} else {
		content.WriteString(styleModalInput.Width(inputWidth).Render(descInput))
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

	// Calculate available lines for content
	maxLines := m.height - 10 // Account for modal padding, title, footer
	if maxLines < 5 {
		maxLines = 5
	}

	totalBindings := len(bindings)
	startIdx := m.helpScrollOffset
	if startIdx > totalBindings-maxLines {
		startIdx = totalBindings - maxLines
	}
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + maxLines
	if endIdx > totalBindings {
		endIdx = totalBindings
	}

	var content strings.Builder
	content.WriteString(styleModalTitle.Render("Keybindings"))
	content.WriteString("\n")

	for i := startIdx; i < endIdx; i++ {
		b := bindings[i]
		content.WriteString(styleHelpKey.Render(fmt.Sprintf("%-12s", b.key)))
		content.WriteString(styleHelpDesc.Render(b.desc))
		if i < endIdx-1 {
			content.WriteString("\n")
		}
	}

	// Scroll indicator
	if totalBindings > maxLines {
		content.WriteString("\n")
		content.WriteString(styleHelpDesc.Render(fmt.Sprintf("(%d-%d of %d) j/k to scroll", startIdx+1, endIdx, totalBindings)))
	}

	content.WriteString("\n")
	content.WriteString(styleHelpDesc.Render("Esc or ? to close"))

	modal := styleModalBox.Render(content.String())
	return centerModal(modal, m.width, m.height)
}

// getHelpBindingsCount returns the number of help bindings for scroll bounds
func (m Model) getHelpBindingsCount() int {
	return 16 // Number of bindings in renderHelpModal
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
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
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

// padToWidth pads or trims a string to exact width
func padToWidth(s string, width int) string {
	runes := []rune(s)
	if len(runes) < width {
		return s + strings.Repeat(" ", width-len(runes))
	}
	if len(runes) > width {
		return string(runes[:width])
	}
	return s
}
