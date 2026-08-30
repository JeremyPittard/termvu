package ui

import (
	"fmt"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	maudio "termvu/musicviz/src/audio"
	mmeta "termvu/musicviz/src/metadata"
)

// MetadataMsg holds updated track metadata.
type MetadataMsg struct {
	Metadata mmeta.Metadata
}

// SpectrumMsg holds updated spectrum data.
type SpectrumMsg struct {
	Spectrum maudio.Spectrum
}

var (
	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)
	artistStyle = lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#884EA0")).
			Padding(0, 1)
	albumStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#00677F")).
			Padding(0, 1)
	barStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#FFFDF5")).
			Width(2)
	noDataStyle = lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#888888"))
)

type model struct {
	metadata mmeta.Metadata
	spectrum maudio.Spectrum
	width    int
	height   int
}

// InitialModel returns the initial UI model.
func InitialModel() model {
	return model{
		metadata: mmeta.Metadata{
			Title:  "No track",
			Artist: "No artist",
			Album:  "No album",
			Playing: false,
		},
		spectrum: maudio.Spectrum{}, // all zeros
	}
}

// Init returns the initial command (nil).
func (m model) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case MetadataMsg:
		m.metadata = msg.Metadata
	case SpectrumMsg:
		m.spectrum = msg.Spectrum
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// View renders the UI.
func (m model) View() tea.View {
	if m.height == 0 || m.width == 0 {
		return tea.NewView("")
	}

	// Header with track info
	header := lipgloss.JoinHorizontal(lipgloss.Center,
		titleStyle.Render(fmt.Sprintf(" %.20s ", m.metadata.Title)),
		artistStyle.Render(fmt.Sprintf(" %.20s ", m.metadata.Artist)),
		albumStyle.Render(fmt.Sprintf(" %.20s ", m.metadata.Album)),
	)

	// Play/pause indicator
	var indicator string
	if m.metadata.Playing {
		indicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Render("▶")
	} else {
		indicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Render("❚❚")
	}
	header = lipgloss.JoinHorizontal(lipgloss.Left, header, " ", indicator)

	// Spectrum bars
	var bars []string
	maxHeight := m.height - 4 // subtract header and padding
	if maxHeight < 1 {
		maxHeight = 1
	}
	for _, amp := range m.spectrum {
		// Normalize amp to 0-1 range (assuming amp is already 0-1 from audio)
		if amp > 1 {
			amp = 1
		}
		if amp < 0 {
			amp = 0
		}
		barHeight := int(amp * float64(maxHeight))
		if barHeight < 1 && amp > 0 {
			barHeight = 1
		}
		// Color based on amplitude (green to red)
		var colorString string
		if amp < 0.3 {
			colorString = "#25A065" // green
		} else if amp < 0.7 {
			colorString = "#FFF44F" // yellow
		} else {
			colorString = "#FF0000" // red
		}
		color := lipgloss.Color(colorString)
		bar := barStyle.Copy().
			Background(color).
			Height(barHeight).
			MarginRight(1).
			String()
		bars = append(bars, bar)
	}
	spectrumView := lipgloss.JoinHorizontal(lipgloss.Top, bars...)

	// Center the spectrum view horizontally
	spectrumContainer := lipgloss.Place(m.width, maxHeight+2, lipgloss.Center, lipgloss.Center, spectrumView)

	// Combine header and spectrum
	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, "\n", spectrumContainer))
}
