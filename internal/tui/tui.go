package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"enva/internal/db"
	"enva/internal/env"
)

// Run starts the TUI application.
func Run(database *db.DB, resolver *env.Resolver, cwd string) error {
	ctx, err := resolver.Resolve(cwd)
	if err != nil {
		return fmt.Errorf("failed to resolve environment: %w", err)
	}

	m := NewModel(database, resolver, ctx)
	p := tea.NewProgram(m, tea.WithAltScreen())

	_, err = p.Run()
	return err
}
