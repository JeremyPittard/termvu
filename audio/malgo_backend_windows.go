//go:build windows

package audio

import (
	"context"
	"fmt"
	"sync"
	"unsafe"

	"github.com/gen2brain/malgo"
)

// MalgoBackend implements CaptureBackend using malgo with WASAPI Loopback
type MalgoBackend struct {
	ctx        *malgo.AllocatedContext
	device     *malgo.Device
	sampleRate uint32
	channels   uint32
	samplesCh  chan []float32
	mu         sync.Mutex
	started    bool
}

// NewMalgoBackend creates a new malgo backend for Windows (WASAPI Loopback)
func NewMalgoBackend() (*MalgoBackend, error) {
	ctx, err := malgo.InitContext([]malgo.Backend{malgo.BackendWasapi}, malgo.ContextConfig{}, func(msg string) {
		// Log callback
	})
	if err != nil {
		return nil, fmt.Errorf("malgo init context (WASAPI): %w", err)
	}

	return &MalgoBackend{
		ctx:       ctx,
		samplesCh: make(chan []float32, 16),
	}, nil
}

// Start opens the WASAPI loopback device and begins capture
func (b *MalgoBackend) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.started {
		return nil
	}

	// WASAPI loopback configuration
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Loopback)
	deviceConfig.Capture.Format = malgo.FormatF32
	deviceConfig.Capture.Channels = 2
	deviceConfig.PeriodSizeInMilliseconds = 10
	deviceConfig.Wasapi.NoAutoConvertSRC = true
	deviceConfig.Wasapi.NoDefaultQualitySRC = true

	deviceCallbacks := malgo.DeviceCallbacks{
		Data: func(pOutputSample, pInputSamples []byte, framecount uint32) {
			floatSamples := unsafe.Slice((*float32)(unsafe.Pointer(&pInputSamples[0])), int(framecount)*int(b.channels))
			if b.channels == 2 && len(floatSamples) >= 2 {
				mono := make([]float32, framecount)
				for i := uint32(0); i < framecount; i++ {
					mono[i] = (floatSamples[i*2] + floatSamples[i*2+1]) * 0.5
				}
				select {
				case b.samplesCh <- mono:
				default:
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
		return fmt.Errorf("init WASAPI capture device: %w", err)
	}

	b.device = device
	b.sampleRate = deviceConfig.SampleRate
	if b.sampleRate == 0 {
		b.sampleRate = 48000
	}
	b.channels = 1

	if err := device.Start(); err != nil {
		return fmt.Errorf("start WASAPI device: %w", err)
	}

	b.started = true
	return nil
}

// Stop stops the capture device
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