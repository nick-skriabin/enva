// Package tui provides the Bubble Tea TUI for enva.
package tui

import "github.com/charmbracelet/lipgloss"

// Use terminal's native ANSI colors (0-15) to inherit from user's theme
var (
	colorNone      = lipgloss.NoColor{}
	colorBlack     = lipgloss.Color("0")
	colorRed       = lipgloss.Color("1")
	colorGreen     = lipgloss.Color("2")
	colorYellow    = lipgloss.Color("3")
	colorBlue      = lipgloss.Color("4")
	colorMagenta   = lipgloss.Color("5")
	colorCyan      = lipgloss.Color("6")
	colorWhite     = lipgloss.Color("7")
	colorBrBlack   = lipgloss.Color("8")  // Bright black (gray)
	colorBrRed     = lipgloss.Color("9")
	colorBrGreen   = lipgloss.Color("10")
	colorBrYellow  = lipgloss.Color("11")
	colorBrBlue    = lipgloss.Color("12")
	colorBrMagenta = lipgloss.Color("13")
	colorBrCyan    = lipgloss.Color("14")
	colorBrWhite   = lipgloss.Color("15")
)

// Styles using terminal colors
var (
	styleAppName = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true)

	styleSearchQuery = lipgloss.NewStyle().
				Foreground(colorYellow).
				Bold(true)

	styleTableHeader = lipgloss.NewStyle().
				Foreground(colorBrBlack).
				Bold(true)

	styleTableRow = lipgloss.NewStyle()

	styleTableRowSelected = lipgloss.NewStyle().
				Background(colorBrBlack).
				Foreground(colorWhite)

	styleBadgeLocal = lipgloss.NewStyle().
			Foreground(colorGreen)

	styleBadgeInherited = lipgloss.NewStyle().
				Foreground(colorCyan)

	styleBadgeOverride = lipgloss.NewStyle().
				Foreground(colorYellow)

	styleBorderTitle = lipgloss.NewStyle().
				Foreground(colorBrBlack)

	styleToast = lipgloss.NewStyle().
			Foreground(colorGreen)

	styleToastError = lipgloss.NewStyle().
			Foreground(colorRed)

	styleMatchHighlight = lipgloss.NewStyle().
				Foreground(colorYellow).
				Bold(true)

	styleModalBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBrBlack).
			Padding(0, 2)

	styleModalTitle = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true)

	styleModalLabel = lipgloss.NewStyle().
			Foreground(colorBrBlack)

	styleModalInput = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBrBlack)

	styleModalInputFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorCyan)

	styleHelpKey = lipgloss.NewStyle().
			Foreground(colorCyan)

	styleHelpDesc = lipgloss.NewStyle().
			Foreground(colorBrBlack)

	styleError = lipgloss.NewStyle().
			Foreground(colorRed)

	styleConfirm = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	styleDim = lipgloss.NewStyle().
			Foreground(colorBrBlack)

	styleCursor = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true)
)

// Badge characters
const (
	badgeLocal     = "●"
	badgeInherited = "○"
	badgeOverride  = "▲"
)
