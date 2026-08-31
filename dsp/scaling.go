package dsp

import (
	"math"

	"termvu/config"
)

// AGCState holds the automatic gain control state
type AGCState struct {
	// Running estimate of peak level
	peak float64
	// Target level in linear scale (derived from AGCTargetDbFS)
	targetLinear float64
	// Attack/release coefficients (per sample at analysis rate)
	attackCoeff  float64
	releaseCoeff float64
	// Per-band multipliers for visual EQ
	bandMultipliers []float64
}

// NewAGCState creates a new AGC state with default parameters
func NewAGCState(numBins int) *AGCState {
	targetLinear := math.Pow(10.0, config.AGCTargetDbFS/20.0)

	// Convert attack/release times to per-analysis-tick coefficients
	// Analysis tick is ~30 Hz (33ms), so we convert ms to per-tick
	attackTicks := float64(config.AGCAttackMs) / float64(config.TickAnalyzeMs)
	releaseTicks := float64(config.AGCReleaseMs) / float64(config.TickAnalyzeMs)

	attackCoeff := 1.0 - math.Exp(-1.0/attackTicks)
	releaseCoeff := 1.0 - math.Exp(-1.0/releaseTicks)

	multipliers := make([]float64, numBins)
	for i := range multipliers {
		multipliers[i] = 1.0
	}

	return &AGCState{
		targetLinear:    targetLinear,
		attackCoeff:     attackCoeff,
		releaseCoeff:    releaseCoeff,
		bandMultipliers: multipliers,
	}
}

// SetBandMultiplier sets the visual EQ multiplier for a specific band
func (a *AGCState) SetBandMultiplier(band int, mult float64) {
	if band >= 0 && band < len(a.bandMultipliers) {
		a.bandMultipliers[band] = mult
	}
}

// SetAllBandMultipliers sets all band multipliers at once
func (a *AGCState) SetAllBandMultipliers(mults []float64) {
	for i := range a.bandMultipliers {
		if i < len(mults) {
			a.bandMultipliers[i] = mults[i]
		}
	}
}

// Process applies dB conversion and AGC to the binned magnitudes.
// Input: binned magnitudes (linear scale)
// Output: normalized 0.0-1.0 values for rendering
func (a *AGCState) Process(binned []float64) []float64 {
	out := make([]float64, len(binned))

	// Find peak across all bands
	maxVal := 0.0
	for i, v := range binned {
		val := v * a.bandMultipliers[i]
		if val > maxVal {
			maxVal = val
		}
	}

	// Update AGC peak estimate
	if maxVal > a.peak {
		// Attack: rise quickly
		a.peak += (maxVal - a.peak) * a.attackCoeff
	} else {
		// Release: decay slowly
		a.peak += (maxVal - a.peak) * a.releaseCoeff
	}

	// Avoid division by zero
	if a.peak < 1e-10 {
		a.peak = 1e-10
	}

	// Compute gain to bring peak to target level
	gain := a.targetLinear / a.peak

	// Apply gain and clamp to 0.0-1.0
	for i, v := range binned {
		val := v * a.bandMultipliers[i] * gain
		if val > 1.0 {
			val = 1.0
		}
		if val < 0.0 {
			val = 0.0
		}
		out[i] = val
	}

	return out
}

// LinearToDb converts linear amplitude to dBFS
func LinearToDb(linear float64) float64 {
	if linear <= 0 {
		return -120.0 // Effectively -infinity
	}
	return 20.0 * math.Log10(linear)
}

// DbToLinear converts dBFS to linear amplitude
func DbToLinear(db float64) float64 {
	return math.Pow(10.0, db/20.0)
}