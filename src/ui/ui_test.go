package ui

import (
	"testing"

	"termvu/musicviz/src/audio"
	"termvu/musicviz/src/metadata"
)

func TestInitialModel(t *testing.T) {
	m := InitialModel()
	if m.metadata.Title == "" {
		t.Error("expected non-empty title")
	}
}

func TestUpdateMetadata(t *testing.T) {
	m := InitialModel()
	meta := metadata.Metadata{Title: "Test", Artist: "Artist", Album: "Album", Playing: true}
	m.Update(MetadataMsg{Metadata: meta})
	if m.metadata.Title != "Test" {
		t.Errorf("expected title Test, got %s", m.metadata.Title)
	}
}

func TestUpdateSpectrum(t *testing.T) {
	m := InitialModel()
	spec := audio.Spectrum{}
	for i := range spec {
		spec[i] = float64(i) / 30.0
	}
	m.Update(SpectrumMsg{Spectrum: spec})
	if m.spectrum[0] != 0 {
		t.Errorf("expected spectrum[0] 0, got %f", m.spectrum[0])
	}
}

func TestViewRenders(t *testing.T) {
	m := InitialModel()
	// Set width and height to avoid early return.
	m.width = 100
	m.height = 20
	v := m.View()
	if v == "" {
		t.Error("expected empty view")
	}
}
