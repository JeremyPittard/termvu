package state

import (
	"sync"
	"time"

	"termvu/dsp"
)

// FrameBuffer provides thread-safe access to the latest DSP frame
type FrameBuffer struct {
	mu         sync.Mutex
	current    dsp.Frame
	updatedAt  time.Time
}

// NewFrameBuffer creates a new frame buffer
func NewFrameBuffer() *FrameBuffer {
	return &FrameBuffer{}
}

// Write stores a new frame
func (b *FrameBuffer) Write(frame dsp.Frame) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.current = frame
	b.updatedAt = time.Now()
}

// Read returns the current frame (zero value if never written)
func (b *FrameBuffer) Read() dsp.Frame {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.current
}

// UpdatedAt returns the time of the last write
func (b *FrameBuffer) UpdatedAt() time.Time {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.updatedAt
}

// HasData returns true if a frame has been written
func (b *FrameBuffer) HasData() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return !b.current.IsEmpty()
}