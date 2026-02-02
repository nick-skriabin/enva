// Package tui provides the Bubble Tea TUI for enva.
package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	colorPrimary    = lipgloss.Color("39")  // Blue
	colorSecondary  = lipgloss.Color("245") // Gray
	colorSuccess    = lipgloss.Color("42")  // Green
	colorWarning    = lipgloss.Color("214") // Orange
	colorError      = lipgloss.Color("196") // Red
	colorHighlight  = lipgloss.Color("226") // Yellow
	colorLocalBadge = lipgloss.Color("42")  // Green
	colorInherited  = lipgloss.Color("245") // Gray
	colorOverride   = lipgloss.Color("214") // Orange
)

// Styles
var (
	styleTopBar = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	styleRoot = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	styleProfile = lipgloss.NewStyle().
			Foreground(colorSecondary)

	styleSearchLabel = lipgloss.NewStyle().
				Foreground(colorSecondary)

	styleSearchQuery = lipgloss.NewStyle().
				Foreground(colorPrimary)

	styleTableHeader = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("238"))

	styleTableRow = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	styleTableRowSelected = lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("252"))

	styleBadgeLocal = lipgloss.NewStyle().
			Foreground(colorLocalBadge)

	styleBadgeInherited = lipgloss.NewStyle().
				Foreground(colorInherited)

	styleBadgeOverride = lipgloss.NewStyle().
				Foreground(colorOverride)

	styleStatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	styleStatusKey = lipgloss.NewStyle().
			Foreground(colorSecondary)

	styleStatusValue = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	styleToast = lipgloss.NewStyle().
			Foreground(colorSuccess)

	styleToastError = lipgloss.NewStyle().
			Foreground(colorError)

	styleMatchHighlight = lipgloss.NewStyle().
				Foreground(colorHighlight).
				Bold(true)

	styleModalBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2)

	styleModalTitle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			MarginBottom(1)

	styleModalLabel = lipgloss.NewStyle().
			Foreground(colorSecondary)

	styleModalInput = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1)

	styleModalInputFocused = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1)

	styleHelpKey = lipgloss.NewStyle().
			Foreground(colorPrimary)

	styleHelpDesc = lipgloss.NewStyle().
			Foreground(colorSecondary)

	styleError = lipgloss.NewStyle().
			Foreground(colorError)

	styleConfirm = lipgloss.NewStyle().
			Foreground(colorWarning).
			Bold(true)
)

// Badge characters
const (
	badgeLocal     = "●" // Local var
	badgeInherited = "○" // Inherited var
	badgeOverride  = "▲" // Overridden (local override of inherited)
)
