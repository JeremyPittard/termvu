package ui

import (
	"time"

	"charm.land/bubbletea/v2"

	"termvu/config"
)

// Tick messages for the bubbletea program
type (
	tickMsg        struct{}
	metadataTickMsg struct{}
)

// TickRenderCmd returns a command that fires at the render tick interval
func TickRenderCmd() tea.Cmd {
	return tea.Tick(config.TickRenderDuration(), func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// TickMetadataCmd returns a command that fires at the metadata tick interval
func TickMetadataCmd() tea.Cmd {
	return tea.Tick(config.TickMetaDuration(), func(t time.Time) tea.Msg {
		return metadataTickMsg{}
	})
}

// TickCmd returns both tick commands
func TickCmd() tea.Cmd {
	return tea.Batch(TickRenderCmd(), TickMetadataCmd())
}