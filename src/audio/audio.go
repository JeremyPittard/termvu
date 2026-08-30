//go:build !windows

package audio

import (
	"context"
	"sync"
	"time"

	"github.com/gordonklaus/portaudio"
)

// AudioCapture captures audio from the default input device and computes the
// spectrum. On non-Windows platforms PortAudio is used (microphone input).
type AudioCapture struct {
	mu         sync.Mutex
	spectrum   Spectrum
	an         *analyzer
	sampleRate float64
	fftSize    int
	stream     *portaudio.Stream
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewAudioCapture creates a new AudioCapture with the given sample rate and FFT size.
func NewAudioCapture(sampleRate float64, fftSize int) (*AudioCapture, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, err
	}

	ac := &AudioCapture{
		an:         newAnalyzer(sampleRate, fftSize),
		sampleRate: sampleRate,
		fftSize:    fftSize,
	}

	stream, err := portaudio.OpenDefaultStream(
		1, // input channels (mono)
		0, // output channels
		sampleRate,
		fftSize, // frames per buffer
		ac.audioCallback,
	)
	if err != nil {
		portaudio.Terminate()
		return nil, err
	}
	ac.stream = stream
	return ac, nil
}

// audioCallback is called by PortAudio for each input buffer.
func (ac *AudioCapture) audioCallback(in []int32) {
	samples := make([]float64, len(in))
	for i, v := range in {
		samples[i] = float64(v) / 2147483647.0
	}

	ac.mu.Lock()
	defer ac.mu.Unlock()
	if s := ac.an.push(samples); s != nil {
		ac.spectrum = *s
	}
}

// Start begins capturing and sends spectrum updates on the returned channel.
// It stops when the context is canceled.
func (ac *AudioCapture) Start(parentCtx context.Context) <-chan Spectrum {
	ac.ctx, ac.cancel = context.WithCancel(parentCtx)

	ch := make(chan Spectrum)
	go func() {
		defer close(ch)
		defer ac.stream.Close()
		defer portaudio.Terminate()

		if err := ac.stream.Start(); err != nil {
			return
		}
		defer ac.stream.Stop()

		ticker := time.NewTicker(50 * time.Millisecond) // ~20 FPS
		defer ticker.Stop()
		for {
			select {
			case <-ac.ctx.Done():
				return
			case <-ticker.C:
				ac.mu.Lock()
				spec := ac.spectrum
				ac.mu.Unlock()
				ch <- spec
			}
		}
	}()
	return ch
}

// Stop stops the audio capture.
func (ac *AudioCapture) Stop() {
	if ac.cancel != nil {
		ac.cancel()
	}
}