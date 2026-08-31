package dsp

import (
	"math"

	"termvu/config"
)

// BinConfig holds configuration for log-frequency binning
type BinConfig struct {
	NumBins   int
	MinFreqHz float64
	MaxFreqHz float64
	SampleRate int
	FFTSize    int
}

// DefaultBinConfig returns the default binning configuration from constants
func DefaultBinConfig(sampleRate int) BinConfig {
	return BinConfig{
		NumBins:    config.NumBins,
		MinFreqHz:  config.MinFreqHz,
		MaxFreqHz:  config.MaxFreqHz,
		SampleRate: sampleRate,
		FFTSize:    config.FFTSize,
	}
}

// LogBinEdges computes the FFT bin indices for each log-frequency visual bin.
// Returns a slice of (start, end) FFT bin index pairs for each visual bin.
func LogBinEdges(cfg BinConfig) [][2]int {
	edges := make([][2]int, cfg.NumBins)

	// Nyquist frequency
	nyquist := float64(cfg.SampleRate) / 2.0
	maxFreq := math.Min(cfg.MaxFreqHz, nyquist)

	// Logarithmic frequency spacing
	logMin := math.Log(cfg.MinFreqHz)
	logMax := math.Log(maxFreq)

	for i := 0; i < cfg.NumBins; i++ {
		// Log-frequency boundaries for this visual bin
		fracStart := float64(i) / float64(cfg.NumBins)
		fracEnd := float64(i+1) / float64(cfg.NumBins)

		freqStart := math.Exp(logMin + fracStart*(logMax-logMin))
		freqEnd := math.Exp(logMin + fracEnd*(logMax-logMin))

		// Convert frequencies to FFT bin indices
		// FFT bin k corresponds to frequency k * sampleRate / FFTSize
		binStart := int(math.Floor(freqStart * float64(cfg.FFTSize) / float64(cfg.SampleRate)))
		binEnd := int(math.Ceil(freqEnd * float64(cfg.FFTSize) / float64(cfg.SampleRate)))

		// Clamp to valid range (0 to FFTSize/2 - 1)
		maxBin := cfg.FFTSize/2 - 1
		if binStart < 0 {
			binStart = 0
		}
		if binStart > maxBin {
			binStart = maxBin
		}
		if binEnd < 0 {
			binEnd = 0
		}
		if binEnd > maxBin+1 {
			binEnd = maxBin + 1
		}
		if binEnd <= binStart {
			binEnd = binStart + 1
		}

		edges[i] = [2]int{binStart, binEnd}
	}

	return edges
}

// BinMagnitudes aggregates FFT magnitude spectrum into log-frequency visual bins.
// magnitude: FFT magnitude spectrum of length FFTSize/2
// edges: output of LogBinEdges
// Returns slice of length NumBins with aggregated energy per visual bin.
func BinMagnitudes(magnitude []float64, edges [][2]int) []float64 {
	numBins := len(edges)
	binned := make([]float64, numBins)

	for i, edge := range edges {
		start, end := edge[0], edge[1]
		if start >= len(magnitude) {
			continue
		}
		if end > len(magnitude) {
			end = len(magnitude)
		}
		sum := 0.0
		count := 0
		for j := start; j < end; j++ {
			sum += magnitude[j]
			count++
		}
		if count > 0 {
			binned[i] = sum / float64(count) // Average magnitude in bin
		}
	}

	return binned
}