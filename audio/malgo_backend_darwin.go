//go:build darwin

package audio

import (
	"context"
	"errors"
	"sync"
)

// MalgoBackend is a stub for macOS - loopback capture requires BlackHole + Multi-Output Device
// See docs/macos_setup.md for setup instructions
type MalgoBackend struct {
	samplesCh chan []float32
	mu        sync.Mutex
	started   bool
}

// NewMalgoBackend creates a stub backend for macOS
func NewMalgoBackend() (*MalgoBackend, error) {
	return &MalgoBackend{
		samplesCh: make(chan []float32, 16),
	}, nil
}

// Start returns an error - macOS loopback requires BlackHole setup
func (b *MalgoBackend) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.started {
		return nil
	}

	return errors.New("macOS loopback capture not implemented in Phase 1. " +
		"Install BlackHole and create a Multi-Output Device in Audio MIDI Setup. " +
		"See docs/macos_setup.md for instructions.")
}

// Stop is a no-op for the stub
func (b *MalgoBackend) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.started {
		return nil
	}

	close(b.samplesCh)
	b.started = false
	return nil
}

// Samples returns a channel that will never receive data (stub)
func (b *MalgoBackend) Samples() <-chan []float32 {
	return b.samplesCh
}

// SampleRate returns the default sample rate
func (b *MalgoBackend) SampleRate() int {
	return 48000
}

// Channels returns 1 (mono)
func (b *MalgoBackend) Channels() int {
	return 1
}