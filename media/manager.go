package media

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// MonitorInterface defines the interface for platform-specific metadata monitors
type MonitorInterface interface {
	UpdateChannel() <-chan Metadata
	Start(context.Context) error
	Stop() error
	Poll()
}

// newPlatformMonitor is implemented in platform-specific files
// manager_linux.go, manager_windows.go, manager_darwin.go

// MetadataManager manages platform-specific metadata sources
type MetadataManager struct {
	mu       sync.RWMutex
	current  Metadata
	updateCh chan Metadata
	stopCh   chan struct{}
	wg       sync.WaitGroup
	monitor  MonitorInterface
}

// NewMetadataManager creates a new metadata manager for the current platform
func NewMetadataManager() (*MetadataManager, error) {
	monitor, err := newPlatformMonitor()
	if err != nil {
		return nil, fmt.Errorf("create metadata monitor for %s: %w", runtime.GOOS, err)
	}

	return &MetadataManager{
		updateCh: make(chan Metadata, 4),
		stopCh:   make(chan struct{}),
		monitor:  monitor,
	}, nil
}

// Start begins metadata monitoring
func (m *MetadataManager) Start(ctx context.Context) error {
	if err := m.monitor.Start(ctx); err != nil {
		return err
	}

	m.wg.Add(1)
	go m.monitorLoop(ctx)

	m.wg.Add(1)
	go m.pollLoop()

	return nil
}

// Stop stops metadata monitoring
func (m *MetadataManager) Stop() error {
	close(m.stopCh)
	m.wg.Wait()
	return m.monitor.Stop()
}

// Current returns the latest metadata
func (m *MetadataManager) Current() Metadata {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// UpdateChannel returns the channel for metadata updates
func (m *MetadataManager) UpdateChannel() <-chan Metadata {
	return m.updateCh
}

// monitorLoop receives updates from the platform monitor
func (m *MetadataManager) monitorLoop(ctx context.Context) {
	defer m.wg.Done()

	ch := m.monitor.UpdateChannel()
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case meta, ok := <-ch:
			if !ok {
				return
			}
			m.mu.Lock()
			m.current = meta
			m.mu.Unlock()

			// Forward to update channel
			select {
			case m.updateCh <- meta:
			default:
			}
		}
	}
}

// pollLoop periodically polls the monitor (for platforms that need it)
func (m *MetadataManager) pollLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.monitor.Poll()
		}
	}
}