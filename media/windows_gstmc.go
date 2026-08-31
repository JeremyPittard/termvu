//go:build windows

package media

import (
	"context"
	"errors"
	"sync"
)

// GSMTCMonitor is a stub for Windows GSMTC metadata
// Phase 1: stub with graceful fallback
// Phase 2+: cgo COM interop implementation
type GSMTCMonitor struct {
	updateCh chan Metadata
	stopCh   chan struct{}
	mu       sync.Mutex
	started  bool
}

// NewGSMTCMonitor creates a new GSMTC monitor (stub)
func NewGSMTCMonitor() (*GSMTCMonitor, error) {
	return &GSMTCMonitor{
		updateCh: make(chan Metadata, 4),
		stopCh:   make(chan struct{}),
	}, nil
}

// UpdateChannel returns the channel that receives metadata updates
func (m *GSMTCMonitor) UpdateChannel() <-chan Metadata {
	return m.updateCh
}

// Start begins monitoring (stub - does nothing)
func (m *GSMTCMonitor) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return nil
	}

	// Phase 1: stub implementation
	// TODO: Implement cgo COM interop for GSMTC in Phase 2
	// See: https://learn.microsoft.com/en-us/windows/win32/api/mediatransportcontrol/nn-mediatransportcontrol-iglobalsystemmediatransportcontrolssessionmanager
	m.started = true
	return nil
}

// Stop stops the monitor
func (m *GSMTCMonitor) Stop() error {
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
func (m *GSMTCMonitor) Poll() {
	// Stub: no metadata available
	// In Phase 2, this would query GSMTC for active session
	select {
	case m.updateCh <- Metadata{}:
	default:
	}
}

// ErrorStub returns an error describing the stub status
func (m *GSMTCMonitor) ErrorStub() error {
	return errors.New("Windows GSMTC metadata not implemented in Phase 1. " +
		"Install a Go GSMTC binding or use fallback label. " +
		"See docs/windows_setup.md for details.")
}