//go:build !windows && !darwin

package audio

import (
	"context"
	"fmt"
	"sync"
	"unsafe"

	"github.com/gen2brain/malgo"
)

// MalgoBackend implements CaptureBackend using malgo (miniaudio)
type MalgoBackend struct {
	ctx        *malgo.AllocatedContext
	device     *malgo.Device
	sampleRate uint32
	channels   uint32
	samplesCh  chan []float32
	mu         sync.Mutex
	started    bool
}

// NewMalgoBackend creates a new malgo backend for Linux (Pulse/PipeWire monitor)
func NewMalgoBackend() (*MalgoBackend, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(msg string) {
		// Log callback - could be configurable
	})
	if err != nil {
		return nil, fmt.Errorf("malgo init context: %w", err)
	}

	return &MalgoBackend{
		ctx:       ctx,
		samplesCh: make(chan []float32, 16),
	}, nil
}

// Start opens the default monitor device and begins capture
func (b *MalgoBackend) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.started {
		return nil
	}

	// Use default device config for loopback
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Loopback)
	deviceConfig.Capture.Format = malgo.FormatF32
	deviceConfig.Capture.Channels = 2 // Stereo
	deviceConfig.PeriodSizeInMilliseconds = 10

	// Create device with callback
	deviceCallbacks := malgo.DeviceCallbacks{
		Data: func(pOutputSample, pInputSamples []byte, framecount uint32) {
			// Convert input bytes to float32
			floatSamples := unsafe.Slice((*float32)(unsafe.Pointer(&pInputSamples[0])), int(framecount)*int(b.channels))
			// Downmix to mono if stereo
			if b.channels == 2 && len(floatSamples) >= 2 {
				mono := make([]float32, framecount)
				for i := uint32(0); i < framecount; i++ {
					mono[i] = (floatSamples[i*2] + floatSamples[i*2+1]) * 0.5
				}
				select {
				case b.samplesCh <- mono:
				default:
					// Drop frame if channel full (backpressure)
				}
			} else {
				select {
				case b.samplesCh <- floatSamples[:framecount]:
				default:
				}
			}
		},
	}

	device, err := malgo.InitDevice(b.ctx.Context, deviceConfig, deviceCallbacks)
	if err != nil {
		return fmt.Errorf("init capture device: %w", err)
	}

	b.device = device
	b.sampleRate = deviceConfig.SampleRate
	if b.sampleRate == 0 {
		b.sampleRate = 48000
	}
	b.channels = 1 // We downmix to mono

	if err := device.Start(); err != nil {
		return fmt.Errorf("start device: %w", err)
	}

	b.started = true
	return nil
}

// Stop stops the capture device and closes the context
func (b *MalgoBackend) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.started {
		return nil
	}

	if b.device != nil {
		b.device.Stop()
		b.device.Uninit()
		b.device = nil
	}

	close(b.samplesCh)
	b.started = false
	return nil
}

// Samples returns the channel delivering captured audio buffers
func (b *MalgoBackend) Samples() <-chan []float32 {
	return b.samplesCh
}

// SampleRate returns the capture sample rate
func (b *MalgoBackend) SampleRate() int {
	return int(b.sampleRate)
}

// Channels returns the number of output channels (always 1 - mono)
func (b *MalgoBackend) Channels() int {
	return 1
}

// Uninit releases the malgo context
func (b *MalgoBackend) Uninit() {
	b.Stop()
	if b.ctx != nil {
		b.ctx.Free()
		b.ctx = nil
	}
}