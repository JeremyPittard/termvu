package audio

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/gordonklaus/portaudio"
	"github.com/madelynnblue/go-dsp/fft"
)

// Spectrum represents the amplitude of 30 frequency bands.
type Spectrum [30]float64

// AudioCapture captures audio from the default output device and computes the FFT.
type AudioCapture struct {
	spectrum     Spectrum
	spectrumMu   sync.Mutex
	sampleRate   float64
	fftSize      int
	audioBuffer  []float64
	window       []float64
	stream       *portaudio.Stream
	ctx          context.Context
	cancelFunc   context.CancelFunc
}

// NewAudioCapture creates a new AudioCapture with given sample rate and FFT size.
func NewAudioCapture(sampleRate float64, fftSize int) (*AudioCapture, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, err
	}

	// Precompute Hann window.
	hann := make([]float64, fftSize)
	for i := range hann {
		hann[i] = 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(fftSize-1)))
	}

	ac := &AudioCapture{
		sampleRate: sampleRate,
		fftSize:    fftSize,
		audioBuffer: make([]float64, 0, fftSize),
		window:     hann,
	}

	var err error
	ac.stream, err = portaudio.OpenDefaultStream(
		1, // input channels (mono)
		0, // output channels
		sampleRate,
		fftSize, // frames per buffer
		ac.audioCallback,
	)
	if err != nil {
		return nil, err
	}

	return ac, nil
}

// audioCallback is called by portaudio for each audio buffer.
func (ac *AudioCapture) audioCallback(in []int32) {
	// Convert int32 to float32 in range [-1, 1].
	for _, v := range in {
		ac.audioBuffer = append(ac.audioBuffer, float64(v)/2147483647.0)
	}

	// Process when we have enough samples.
	if len(ac.audioBuffer) >= ac.fftSize {
		// Apply window.
		windowed := make([]float64, ac.fftSize)
		copy(windowed, ac.audioBuffer[:ac.fftSize])
		for i := range windowed {
			windowed[i] *= ac.window[i]
		}

		// Compute FFT using FFTReal.
		complexSpectrum := fft.FFTReal(windowed)
		// Compute magnitude spectrum for positive frequencies (including Nyquist).
		mag := make([]float64, ac.fftSize/2+1)
		for i := 0; i < ac.fftSize/2+1; i++ {
			c := complexSpectrum[i]
			mag[i] = math.Hypot(real(c), imag(c))
		}

		// Accumulate magnitudes into bands.
		nyquist := ac.sampleRate / 2
		bandWidth := nyquist / float64(len(ac.spectrum)) // frequency width per band
		var bandSums [30]float64
		var bandCounts [30]int
		binWidth := ac.sampleRate / float64(ac.fftSize) // frequency per FFT bin
		for binIdx, magnitude := range mag {
			freq := float64(binIdx) * binWidth
			bandIdx := int(freq / bandWidth)
			if bandIdx >= len(ac.spectrum) {
				bandIdx = len(ac.spectrum) - 1
			}
			bandSums[bandIdx] += magnitude
			bandCounts[bandIdx]++
		}
		// Compute average for each band.
		for i := range ac.spectrum {
			if bandCounts[i] > 0 {
				ac.spectrum[i] = bandSums[i] / float64(bandCounts[i])
			} else {
				ac.spectrum[i] = 0
			}
		}

		ac.spectrumMu.Lock()
		// ac.spectrum already updated
		ac.spectrumMu.Unlock()

		// Remove processed samples.
		ac.audioBuffer = ac.audioBuffer[ac.fftSize:]
	}
}

// Start begins capturing audio and sends spectrum updates on the returned channel.
// It stops when the context is canceled.
func (ac *AudioCapture) Start(parentCtx context.Context) <-chan Spectrum {
	ac.ctx, ac.cancelFunc = context.WithCancel(parentCtx)

	ch := make(chan Spectrum)
	go func() {
		defer close(ch)
		defer ac.stream.Close()
		defer portaudio.Terminate()

		if err := ac.stream.Start(); err != nil {
			return
		}
		defer ac.stream.Stop()

		ticker := time.NewTicker(50 * time.Millisecond) // 20 FPS
		defer ticker.Stop()
		for {
			select {
			case <-ac.ctx.Done():
				return
			case <-ticker.C:
				ac.spectrumMu.Lock()
				spec := ac.spectrum
				ac.spectrumMu.Unlock()
				ch <- spec
			}
		}
	}()
	return ch
}

// Stop stops the audio capture.
func (ac *AudioCapture) Stop() {
	if ac.cancelFunc != nil {
		ac.cancelFunc()
	}
}
