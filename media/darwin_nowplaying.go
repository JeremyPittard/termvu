//go:build darwin

package media

import (
	"context"
	"errors"
	"sync"
)

// NowPlayingMonitor is a stub for macOS NowPlaying metadata
// Phase 1: stub only - loopback capture requires BlackHole setup
type NowPlayingMonitor struct {
	updateCh chan Metadata
	stopCh   chan struct{}
	mu       sync.Mutex
	started  bool
}

// NewNowPlayingMonitor creates a new NowPlaying monitor (stub)
func NewNowPlayingMonitor() (*NowPlayingMonitor, error) {
	return &NowPlayingMonitor{
		updateCh: make(chan Metadata, 4),
		stopCh:   make(chan struct{}),
	}, nil
}

// UpdateChannel returns the channel that receives metadata updates
func (m *NowPlayingMonitor) UpdateChannel() <-chan Metadata {
	return m.updateCh
}

// Start begins monitoring (stub - does nothing)
func (m *NowPlayingMonitor) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return nil
	}

	// Phase 1: stub implementation
	// Loopback capture requires BlackHole + Multi-Output Device
	// See docs/macos_setup.md
	m.started = true
	return nil
}

// Stop stops the monitor
func (m *NowPlayingMonitor) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return nil
	}

	close(m.stopCh)
	close(m.updateCh)
	m.started = false
	return nil
}

// Poll is called periodically - stub returns no metadata
func (m *NowPlayingMonitor) Poll() {
	select {
	case m.updateCh <- Metadata{}:
	default:
	}
}

// ErrorStub returns an error describing the stub status
func (m *NowPlayingMonitor) ErrorStub() error {
	return errors.New("macOS NowPlaying metadata not implemented in Phase 1. " +
		"Loopback capture requires BlackHole + Multi-Output Device. " +
		"See docs/macos_setup.md for setup instructions.")
}