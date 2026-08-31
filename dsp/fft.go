package dsp

import (
	"math"
	"sync"
)

// FFT holds precomputed twiddle factors for a given size
type FFT struct {
	size     int
	twiddles []complex128
	rev      []int
	once     sync.Once
}

// NewFFT creates an FFT processor for the given size (must be power of 2)
func NewFFT(size int) *FFT {
	if size&(size-1) != 0 {
		panic("FFT size must be a power of 2")
	}
	return &FFT{size: size}
}

// initTwiddles precomputes twiddle factors and bit-reversal indices
func (f *FFT) initTwiddles() {
	f.twiddles = make([]complex128, f.size)
	f.rev = make([]int, f.size)

	// Precompute twiddle factors
	for i := 0; i < f.size; i++ {
		f.twiddles[i] = complex(math.Cos(2*math.Pi*float64(i)/float64(f.size)),
			-math.Sin(2*math.Pi*float64(i)/float64(f.size)))
	}

	// Precompute bit-reversal permutation
	log2n := 0
	for (1 << log2n) < f.size {
		log2n++
	}
	for i := 0; i < f.size; i++ {
		rev := 0
		for j := 0; j < log2n; j++ {
			if i&(1<<j) != 0 {
				rev |= 1 << (log2n - 1 - j)
			}
		}
		f.rev[i] = rev
	}
}

// Transform performs in-place complex FFT on the input buffer.
// Input must have length equal to FFT size.
// Real and imaginary parts are interleaved: [re0, im0, re1, im1, ...]
func (f *FFT) Transform(data []complex128) {
	f.once.Do(f.initTwiddles)

	if len(data) != f.size {
		panic("FFT input size must match FFT size")
	}

	// Bit-reversal permutation
	for i := 0; i < f.size; i++ {
		if f.rev[i] > i {
			data[i], data[f.rev[i]] = data[f.rev[i]], data[i]
		}
	}

	// Cooley-Tukey iterative FFT
	for len := 2; len <= f.size; len <<= 1 {
		half := len >> 1
		step := f.size / len
		for i := 0; i < f.size; i += len {
			for j := 0; j < half; j++ {
				u := data[i+j]
				v := data[i+j+half] * f.twiddles[j*step]
				data[i+j] = u + v
				data[i+j+half] = u - v
			}
		}
	}
}

// TransformReal performs FFT on real input, returning magnitude spectrum (first half only).
// Input: real samples (float32 or float64 slice of length FFTSize)
// Output: magnitude spectrum of length FFTSize/2
func (f *FFT) TransformReal(samples []float32) []float64 {
	f.once.Do(f.initTwiddles)

	if len(samples) != f.size {
		panic("FFT input size must match FFT size")
	}

	// Convert to complex buffer
	data := make([]complex128, f.size)
	for i, s := range samples {
		data[i] = complex(float64(s), 0)
	}

	// Bit-reversal permutation
	for i := 0; i < f.size; i++ {
		if f.rev[i] > i {
			data[i], data[f.rev[i]] = data[f.rev[i]], data[i]
		}
	}

	// Cooley-Tukey iterative FFT
	for len := 2; len <= f.size; len <<= 1 {
		half := len >> 1
		step := f.size / len
		for i := 0; i < f.size; i += len {
			for j := 0; j < half; j++ {
				u := data[i+j]
				v := data[i+j+half] * f.twiddles[j*step]
				data[i+j] = u + v
				data[i+j+half] = u - v
			}
		}
	}

	// Return magnitude spectrum (first half only, since input is real)
	mag := make([]float64, f.size/2)
	for i := 0; i < f.size/2; i++ {
		mag[i] = complexAbs(data[i])
	}
	return mag
}

func complexAbs(c complex128) float64 {
	return math.Hypot(real(c), imag(c))
}

// MagnitudeSpectrum computes the magnitude spectrum from real samples.
// This is a convenience function that creates an FFT, applies Hann window, and returns magnitudes.
func MagnitudeSpectrum(samples []float32, fft *FFT) []float64 {
	// Apply Hann window in-place
	HannWindow(samples)
	return fft.TransformReal(samples)
}