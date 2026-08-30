package audio

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/gordonklaus/portaudio"
	"github.com/mjibson/go-dsp/fft"
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
	fftHelper    *fft.FFT
	stream       *portaudio.Stream
	ctx          context.Context
	cancelFunc   context.CancelFunc
}

// NewAudioCapture creates a new AudioCapture with given sample rate and FFT size.
func NewAudioCapture(sampleRate float64, fftSize int) (*AudioCapture, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, err
	}

	hann := make([]float64, fftSize)
	for i := range hann {
		hann[i] = 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(fftSize-1)))
	}

	ac := &AudioCapture{
		sampleRate: sampleRate,
		fftSize:    fftSize,
		audioBuffer: make([]float64, 0, fftSize),
		window:     hann,
		fftHelper:  fft.NewFFT(fftSize),
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

		// Compute FFT.
		fftHelper := ac.fftHelper
		fftHelper.CopyComplex128s(windowed)
		fftHelper.Transform()

		// Compute magnitude spectrum.
		mag := make([]float64, ac.fftSize/2+1) // only positive frequencies
		for i := 0; i <= ac.fftSize/2; i++ {
			re := fftHelper.OUT[i*2]
			im := fftHelper.OUT[i*2+1]
			mag[i] = math.Hypot(re, im)
		}

		// Convert to dB? We'll just use magnitude.
		// Split into bands.
		spectrum := Spectrum{}
		nyquist := ac.sampleRate / 2
		bandWidth := nyquist / float64(len(spectrum))
		for i := range spectrum {
			low := float64(i) * bandWidth
			high := float64(i+1) * bandWidth
			// Sum magnitudes in this band.
			var sum float64
			var count int
			for j, f := float64(0), 0; j < nyquist && f < len(mag); j += ac.sampleRate / float64(ac.fftSize) {
				if f >= low && f < high {
					sum += mag[f]
					count++
				}
				f += ac.sampleRate / float64(ac.fftSize)
			}
			var avg float64
			if count > 0 {
				avg = sum / float64(count)
			}
			// Normalize by dividing by (fftSize/2) maybe? We'll keep as is for now.
			spectrum[i] = avg
		}

		ac.spectrumMu.Lock()
		ac.spectrum = spectrum
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
