package dsp

import "math"

// HannWindow applies a Hann window to the input buffer in-place.
// The Hann window reduces spectral leakage in FFT analysis.
func HannWindow(buf []float32) {
	n := len(buf)
	if n == 0 {
		return
	}
	for i := range buf {
		// Hann window: 0.5 * (1 - cos(2*pi*i/(n-1)))
		buf[i] *= float32(0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(i)/float64(n-1))))
	}
}

// HannWindowFloat64 applies a Hann window to a float64 buffer in-place.
func HannWindowFloat64(buf []float64) {
	n := len(buf)
	if n == 0 {
		return
	}
	for i := range buf {
		buf[i] *= 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(i)/float64(n-1)))
	}
}