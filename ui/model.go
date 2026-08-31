package ui

import (
	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"termvu/state"
)

// Model is the main TUI model
type Model struct {
	frameBuffer *state.FrameBuffer
	metaStore   *state.MetadataStore
	width       int
	height      int
	styles      Styles
	visualizer  *Visualizer
	quitting    bool
}

// NewModel creates a new TUI model
func NewModel(frameBuffer *state.FrameBuffer, metaStore *state.MetadataStore) *Model {
	return &Model{
		frameBuffer: frameBuffer,
		metaStore:   metaStore,
		styles:      NewStyles(),
		visualizer:  NewVisualizer(),
	}
}

// Init initializes the model (returns initial commands)
func (m *Model) Init() tea.Cmd {
	return TickCmd()
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		return m, TickRenderCmd()
	case metadataTickMsg:
		return m, TickMetadataCmd()
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.visualizer.SetWidth(msg.Width)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the UI
func (m *Model) View() tea.View {
	if m.quitting {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}

	if m.width == 0 || m.height == 0 {
		v := tea.NewView(m.styles.TooSmall.Render("Terminal too small"))
		v.AltScreen = true
		return v
	}

	frame := m.frameBuffer.Read()
	meta := m.metaStore.Read()

	// Build the view
	header := m.renderHeader(meta)
	spectrum := m.visualizer.Render(frame.Values, m.width, m.height-2) // -2 for header

	content := lipgloss.JoinVertical(lipgloss.Left, header, spectrum)

	v := tea.NewView(content)
	v.AltScreen = true
	v.BackgroundColor = m.styles.Background
	v.ForegroundColor = m.styles.Foreground
	return v
}

func (m *Model) renderHeader(meta state.Metadata) string {
	title := meta.DisplayTitle()
	subtitle := meta.DisplaySubtitle()

	titleStyled := m.styles.Title.Render(title)
	subtitleStyled := m.styles.Subtitle.Render(subtitle)

	header := lipgloss.JoinVertical(lipgloss.Left, titleStyled, subtitleStyled)
	return m.styles.HeaderBox.Render(header)
}