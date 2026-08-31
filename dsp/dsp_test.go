package dsp

import (
	"testing"
)

func TestHannWindow(t *testing.T) {
	buf := make([]float32, 10)
	for i := range buf {
		buf[i] = 1.0
	}
	HannWindow(buf)

	// First and last should be 0 (or very close)
	if buf[0] > 0.001 {
		t.Errorf("HannWindow[0] = %f, expected ~0", buf[0])
	}
	if buf[9] > 0.001 {
		t.Errorf("HannWindow[9] = %f, expected ~0", buf[9])
	}

	// Middle should be close to 1.0 (for n=10, max is at i=4,5 with value ~0.97)
	if buf[4] < 0.95 || buf[4] > 1.0 {
		t.Errorf("HannWindow[4] = %f, expected ~0.97", buf[4])
	}
	if buf[5] < 0.95 || buf[5] > 1.0 {
		t.Errorf("HannWindow[5] = %f, expected ~0.97", buf[5])
	}
}

func TestHannWindowFloat64(t *testing.T) {
	buf := make([]float64, 10)
	for i := range buf {
		buf[i] = 1.0
	}
	HannWindowFloat64(buf)

	if buf[0] > 0.001 {
		t.Errorf("HannWindowFloat64[0] = %f, expected ~0", buf[0])
	}
	if buf[9] > 0.001 {
		t.Errorf("HannWindowFloat64[9] = %f, expected ~0", buf[9])
	}
}

func TestFFT(t *testing.T) {
	fft := NewFFT(8)

	// Test with a signal that has DC component
	samples := make([]float32, 8)
	for i := range samples {
		samples[i] = 1.0 // Constant signal = DC
	}

	mag := fft.TransformReal(samples)

	if len(mag) != 4 {
		t.Errorf("Expected 4 magnitude bins, got %d", len(mag))
	}

	// DC component (bin 0) should be non-zero for constant signal
	if mag[0] == 0 {
		t.Error("Expected non-zero DC component")
	}
}

func TestFFTSizeMustBePowerOfTwo(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for non-power-of-2 FFT size")
		}
	}()
	NewFFT(100)
}

func TestMagnitudeSpectrum(t *testing.T) {
	fft := NewFFT(8)
	samples := make([]float32, 8)
	for i := range samples {
		samples[i] = 1.0
	}

	mag := MagnitudeSpectrum(samples, fft)

	if len(mag) != 4 {
		t.Errorf("Expected 4 magnitude bins, got %d", len(mag))
	}

	// DC should be non-zero
	if mag[0] == 0 {
		t.Error("Expected non-zero DC after Hann window")
	}
}

func TestLogBinEdges(t *testing.T) {
	cfg := DefaultBinConfig(48000)
	edges := LogBinEdges(cfg)

	if len(edges) != cfg.NumBins {
		t.Errorf("Expected %d bin edges, got %d", cfg.NumBins, len(edges))
	}

	// Check edges are monotonic
	for i := 1; i < len(edges); i++ {
		if edges[i][0] < edges[i-1][0] {
			t.Errorf("Bin edges not monotonic at %d: %v < %v", i, edges[i], edges[i-1])
		}
	}

	// Check all bins have valid ranges
	for i, e := range edges {
		if e[1] <= e[0] {
			t.Errorf("Bin %d has invalid range: %v", i, e)
		}
		if e[1] > cfg.FFTSize/2 {
			t.Errorf("Bin %d exceeds Nyquist: %v", i, e)
		}
	}
}

func TestBinMagnitudes(t *testing.T) {
	cfg := DefaultBinConfig(48000)
	edges := LogBinEdges(cfg)

	// Create a magnitude spectrum with energy in specific bins
	mag := make([]float64, cfg.FFTSize/2)
	mag[10] = 1.0 // Low frequency
	mag[100] = 1.0 // Mid frequency
	mag[200] = 1.0 // High frequency

	binned := BinMagnitudes(mag, edges)

	if len(binned) != cfg.NumBins {
		t.Errorf("Expected %d binned values, got %d", cfg.NumBins, len(binned))
	}

	// Should have some non-zero values
	nonZero := 0
	for _, v := range binned {
		if v > 0 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Error("Expected some non-zero binned values")
	}
}

func TestAGCState(t *testing.T) {
	agc := NewAGCState(10)

	// Test with silence
	silence := make([]float64, 10)
	out := agc.Process(silence)
	for _, v := range out {
		if v != 0 {
			t.Errorf("Expected 0 for silence, got %f", v)
		}
	}

	// Test with signal
	signal := make([]float64, 10)
	for i := range signal {
		signal[i] = 0.5
	}
	out = agc.Process(signal)

	// Values should be scaled to target
	for _, v := range out {
		if v < 0 || v > 1.0 {
			t.Errorf("Output out of range [0,1]: %f", v)
		}
	}
}

func TestLinearToDb(t *testing.T) {
	tests := []struct {
		linear      float64
		expectedDb  float64
	}{
		{1.0, 0.0},
		{0.5, -6.02}, // approx
		{0.1, -20.0},
		{0.01, -40.0},
	}

	for _, tc := range tests {
		db := LinearToDb(tc.linear)
		if db < tc.expectedDb-0.1 || db > tc.expectedDb+0.1 {
			t.Errorf("LinearToDb(%f) = %f, expected %f", tc.linear, db, tc.expectedDb)
		}
	}
}

func TestDbToLinear(t *testing.T) {
	tests := []struct {
		db             float64
		expectedLinear float64
	}{
		{0.0, 1.0},
		{-6.02, 0.5}, // approx
		{-20.0, 0.1},
		{-40.0, 0.01},
	}

	for _, tc := range tests {
		linear := DbToLinear(tc.db)
		if linear < tc.expectedLinear*0.99 || linear > tc.expectedLinear*1.01 {
			t.Errorf("DbToLinear(%f) = %f, expected %f", tc.db, linear, tc.expectedLinear)
		}
	}
}
