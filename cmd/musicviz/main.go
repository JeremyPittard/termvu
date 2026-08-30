package main

import (
	"context"
	"log"
	"time"

	"termvu/musicviz/src/audio"
	"termvu/musicviz/src/metadata"
	"termvu/musicviz/src/ui"

	"charm.land/bubbletea/v2"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up metadata polling
	metaCh := make(chan metadata.Metadata)
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			meta, err := metadata.GetMetadata()
			if err != nil {
				log.Printf("metadata error: %v", err)
				continue
			}
			select {
			case <-ctx.Done():
				return
			case metaCh <- meta:
			case <-ticker.C:
			}
		}
	}()

	// Set up audio capture
	specCh := make(chan audio.Spectrum)
	ac, err := audio.NewAudioCapture(44100, 1024)
	if err != nil {
		log.Fatalf("failed to create audio capture: %v", err)
	}
	defer ac.Stop()
	go func() {
		for spec := range ac.Start(ctx) {
			select {
			case <-ctx.Done():
				return
			case specCh <- spec:
			}
		}
	}()

	// Create initial model
	model := ui.InitialModel()
	p := tea.NewProgram(model)

	// Goroutine to forward metadata and spectrum to the tea program
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case meta := <-metaCh:
				p.Send(ui.MetadataMsg{Metadata: meta})
			case spec := <-specCh:
				p.Send(ui.SpectrumMsg{Spectrum: spec})
			}
		}
	}()

	if _, err := p.Run(); err != nil {
		log.Fatalf("tea program error: %v", err)
	}
}
