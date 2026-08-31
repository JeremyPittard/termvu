package ui

import (
	"image/color"
	"math"

	"charm.land/lipgloss/v2"
)

// Visualizer renders the spectrum bars
type Visualizer struct {
	width       int
	maxHeight   int
	blockChars  []string
	colorStops  []color.Color
}

// NewVisualizer creates a new visualizer
func NewVisualizer() *Visualizer {
	// Unicode block characters for different fill levels (bottom to top)
	// Using lower/upper half blocks for smooth gradients
	blockChars := []string{
		" ",      // 0%
		"▁",      // 1/8
		"▂",      // 2/8
		"▃",      // 3/8
		"▄",      // 4/8
		"▅",      // 5/8
		"▆",      // 6/8
		"▇",      // 7/8
		"█",      // 8/8 (full)
	}

	// Color gradient from blue -> purple -> pink (matching Tokyo Night)
	colorStops := []color.Color{
		lipgloss.Color("#7aa2f7"), // blue
		lipgloss.Color("#9ece6a"), // green
		lipgloss.Color("#e0af68"), // yellow
		lipgloss.Color("#f7768e"), // red/pink
	}

	return &Visualizer{
		blockChars: blockChars,
		colorStops: colorStops,
	}
}

// SetWidth updates the visualizer width
func (v *Visualizer) SetWidth(w int) {
	v.width = w
	v.maxHeight = w // Will be constrained by terminal height in Render
}

// Render renders the spectrum as vertical bars
func (v *Visualizer) Render(values []float64, termWidth, termHeight int) string {
	if len(values) == 0 {
		return v.renderEmpty(termWidth, termHeight)
	}

	numBins := len(values)
	barWidth := termWidth / numBins
	if barWidth < 1 {
		barWidth = 1
	}

	// Available height for spectrum (leave room for header)
	maxBarHeight := termHeight - 2
	if maxBarHeight < 1 {
		maxBarHeight = 1
	}
	v.maxHeight = maxBarHeight

	var lines []string
	for row := maxBarHeight - 1; row >= 0; row-- {
		var line string
		for _, val := range values {
			bar := v.renderBar(val, barWidth, row)
			line += bar
		}
		lines = append(lines, line)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderBar renders a single vertical bar at a specific row
func (v *Visualizer) renderBar(value float64, width, row int) string {
	// value is 0.0-1.0, map to bar height
	barHeight := int(math.Round(value * float64(v.maxHeight)))

	if row < barHeight {
		// This row is within the bar - choose fill character based on position
		fillRatio := float64(row) / float64(barHeight)
		charIdx := int(fillRatio * float64(len(v.blockChars)-1))
		if charIdx >= len(v.blockChars) {
			charIdx = len(v.blockChars) - 1
		}
		char := v.blockChars[charIdx]

		// Color based on height (taller = hotter color)
		colorIdx := int(float64(row) / float64(v.maxHeight) * float64(len(v.colorStops)-1))
		if colorIdx >= len(v.colorStops) {
			colorIdx = len(v.colorStops) - 1
		}
		clr := v.colorStops[colorIdx]

		style := lipgloss.NewStyle().Foreground(clr)
		if row == barHeight-1 {
			// Peak - use peak style
			style = style.Bold(true)
		}

		return style.Render(char)
	}

	// Empty space
	return v.emptyStyle().Render(" ")
}

func (v *Visualizer) emptyStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#24283b"))
}

func (v *Visualizer) renderEmpty(width, height int) string {
	empty := v.emptyStyle().Render(" ")
	line := lipgloss.NewStyle().Width(width).Render(empty)
	var lines []string
	for i := 0; i < height; i++ {
		lines = append(lines, line)
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}