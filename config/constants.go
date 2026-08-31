package config

import "time"

const (
	// SampleRate is the target sample rate. Actual rate comes from the audio device.
	SampleRate = 48000

	// CapturePeriodMs is the capture period in milliseconds.
	// ~10ms @ 48kHz = ~480 frames per buffer
	CapturePeriodMs = 10

	// CaptureBufferFrames is the maximum buffer size (power of 2 for FFT)
	CaptureBufferFrames = 1024

	// FFTSize is the FFT size (must be power of 2)
	FFTSize = 1024

	// NumBins is the number of visual spectrum columns
	NumBins = 60

	// MinFreqHz is the minimum frequency for log-frequency binning
	MinFreqHz = 30.0

	// MaxFreqHz is the maximum frequency for log-frequency binning
	MaxFreqHz = 16000.0

	// TickRenderMs is the render tick interval (60 FPS = 16ms)
	TickRenderMs = 16

	// TickAnalyzeMs is the FFT analysis tick interval (~30 Hz = 33ms)
	TickAnalyzeMs = 33

	// TickMetaMs is the metadata poll interval
	TickMetaMs = 1500

	// AGCTargetDbFS is the AGC target level in dBFS
	AGCTargetDbFS = -6.0

	// AGCAttackMs is the AGC attack time in milliseconds
	AGCAttackMs = 100

	// AGCReleaseMs is the AGC release time in milliseconds
	AGCReleaseMs = 500
)

// TickRenderDuration returns the render tick duration
func TickRenderDuration() time.Duration {
	return time.Duration(TickRenderMs) * time.Millisecond
}

// TickAnalyzeDuration returns the analysis tick duration
func TickAnalyzeDuration() time.Duration {
	return time.Duration(TickAnalyzeMs) * time.Millisecond
}

// TickMetaDuration returns the metadata tick duration
func TickMetaDuration() time.Duration {
	return time.Duration(TickMetaMs) * time.Millisecond
}