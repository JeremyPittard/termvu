package audio

import (
	"math"

	"github.com/madelynnblue/go-dsp/fft"
)

// numBands is the number of frequency bands in a Spectrum.
const numBands = 30

// Spectrum represents the amplitude of numBands frequency bands.
type Spectrum [numBands]float64

// analyzer accumulates samples, applies a Hann window, computes a real FFT,
// and folds magnitudes into numBands linearly spaced frequency bands.
type analyzer struct {
	sampleRate float64
	fftSize    int
	window     []float64
	buffer     []float64
}

func newAnalyzer(sampleRate float64, fftSize int) *analyzer {
	hann := make([]float64, fftSize)
	for i := range hann {
		hann[i] = 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(fftSize-1)))
	}
	return &analyzer{
		sampleRate: sampleRate,
		fftSize:    fftSize,
		window:     hann,
		buffer:     make([]float64, 0, fftSize),
	}
}

// push appends samples and returns a *Spectrum when a full FFT frame is ready,
// otherwise nil. The caller keeps ownership of the returned pointer or copies
// it immediately; subsequent pushes may reuse internal storage.
func (a *analyzer) push(samples []float64) *Spectrum {
	a.buffer = append(a.buffer, samples...)
	if len(a.buffer) < a.fftSize {
		return nil
	}

	frame := a.buffer[:a.fftSize]
	a.buffer = append([]float64(nil), a.buffer[a.fftSize:]...)

	windowed := make([]float64, a.fftSize)
	for i := range windowed {
		windowed[i] = frame[i] * a.window[i]
	}

	complexSpectrum := fft.FFTReal(windowed)

	// Magnitude of positive frequencies (including Nyquist).
	mag := make([]float64, a.fftSize/2+1)
	for i := range mag {
		c := complexSpectrum[i]
		mag[i] = math.Hypot(real(c), imag(c))
	}

	nyquist := a.sampleRate / 2
	bandWidth := nyquist / numBands
	binWidth := a.sampleRate / float64(a.fftSize)

	var bandSums [numBands]float64
	var bandCounts [numBands]int
	for binIdx, m := range mag {
		freq := float64(binIdx) * binWidth
		bandIdx := int(freq / bandWidth)
		if bandIdx >= numBands {
			bandIdx = numBands - 1
		}
		bandSums[bandIdx] += m
		bandCounts[bandIdx]++
	}

	var spec Spectrum
	for i := range spec {
		if bandCounts[i] > 0 {
			spec[i] = bandSums[i] / float64(bandCounts[i])
		}
	}
	return &spec
}