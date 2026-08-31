package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Styles holds all lipgloss styles
type Styles struct {
	Background  color.Color
	Foreground  color.Color
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	HeaderBox   lipgloss.Style
	TooSmall    lipgloss.Style
	SpectrumBar lipgloss.Style
	SpectrumPeak lipgloss.Style
	SpectrumEmpty lipgloss.Style
}

// NewStyles creates the default styles
func NewStyles() Styles {
	bgColor := lipgloss.Color("#1a1b26")
	fgColor := lipgloss.Color("#c0caf5")

	return Styles{
		Background:  bgColor,
		Foreground:  fgColor,
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7aa2f7")).
			MarginBottom(0),
		Subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565f89")).
			MarginTop(0),
		HeaderBox: lipgloss.NewStyle().
			Padding(0, 1).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3b4261")).
			Width(0), // Auto width
		TooSmall: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f7768e")).
			Bold(true).
			Align(lipgloss.Center),
		SpectrumBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7aa2f7")),
		SpectrumPeak: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#bb9af7")).
			Bold(true),
		SpectrumEmpty: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#24283b")),
	}
}