package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "charm.land/bubbletea/v2"

	"termvu/audio"
	"termvu/config"
	"termvu/dsp"
	"termvu/media"
	"termvu/state"
	"termvu/ui"
)

var (
	backendFlag = flag.String("backend", "malgo", "Audio backend: malgo")
	deviceFlag  = flag.String("device", "auto", "Capture device: auto or device ID")
	binsFlag    = flag.Int("bins", config.NumBins, "Spectrum columns")
	fpsFlag     = flag.Int("fps", 60, "Render FPS")
	helpFlag    = flag.Bool("help", false, "Show help")
)

func main() {
	flag.Parse()

	if *helpFlag {
		printUsage()
		return
	}

	// Create shared state
	frameBuffer := state.NewFrameBuffer()
	metaStore := state.NewMetadataStore()

	// Create audio backend
	var backend audio.CaptureBackend
	var err error

	switch *backendFlag {
	case "malgo":
		backend, err = audio.NewMalgoBackend()
	default:
		fmt.Fprintf(os.Stderr, "Unknown backend: %s\n", *backendFlag)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create audio backend: %v\n", err)
		os.Exit(1)
	}

	// Create metadata manager
	metaManager, err := media.NewMetadataManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create metadata manager: %v\n", err)
		os.Exit(1)
	}

	// Create DSP pipeline
	pipeline := dsp.NewPipeline(config.SampleRate)

	// Use bins flag to configure visualizer (passed via config)
	_ = *binsFlag // TODO: pass to visualizer config
	_ = *deviceFlag // TODO: pass to backend config

	// Channels for audio flow
	samplesCh := make(chan []float32, 32)
	framesCh := make(chan dsp.Frame, 32)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Start audio capture
	if err := backend.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start audio capture: %v\n", err)
		os.Exit(1)
	}

	// Start metadata manager
	if err := metaManager.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start metadata manager: %v\n", err)
	}

	// Start DSP pipeline goroutine
	go pipeline.Run(ctx, samplesCh, framesCh)

	// Start audio -> samples forwarding goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case samples, ok := <-backend.Samples():
				if !ok {
					return
				}
				select {
				case samplesCh <- samples:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// Start frames -> frameBuffer goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case frame, ok := <-framesCh:
				if !ok {
					return
				}
				frameBuffer.Write(frame)
			}
		}
	}()

	// Start metadata -> metaStore goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case meta, ok := <-metaManager.UpdateChannel():
				if !ok {
					return
				}
				metaStore.Write(meta)
			}
		}
	}()

	// Create and run bubbletea program
	model := ui.NewModel(frameBuffer, metaStore)
	prog := tea.NewProgram(
		model,
		tea.WithFPS(*fpsFlag),
	)

	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}

	// Cleanup
	backend.Stop()
	metaManager.Stop()
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `termvu - System-wide terminal audio visualizer

Usage:
  termvu [flags]

Flags:
  -backend string   Audio backend (default "malgo")
  -device string    Capture device (default "auto")
  -bins int         Spectrum columns (default %d)
  -fps int          Render FPS (default 60)
  -help             Show this help

`, config.NumBins)
}