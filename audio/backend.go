package audio

import (
	"context"
)

// CaptureBackend defines the interface for audio capture backends
type CaptureBackend interface {
	// Start begins capturing audio. Returns error if device cannot be opened.
	Start(ctx context.Context) error

	// Stop stops capturing and releases the device.
	Stop() error

	// Samples returns a channel that delivers captured PCM buffers.
	// Each buffer is a slice of float32 samples (interleaved channels).
	Samples() <-chan []float32

	// SampleRate returns the actual sample rate of the capture device.
	SampleRate() int

	// Channels returns the number of audio channels (1 for mono, 2 for stereo).
	Channels() int
}