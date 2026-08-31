package dsp

import (
	"context"
	"time"

	"termvu/config"
)

// Pipeline runs the DSP processing in a background goroutine
type Pipeline struct {
	fft        *FFT
	binEdges   [][2]int
	agc        *AGCState
	sampleRate int
}

// NewPipeline creates a new DSP pipeline
func NewPipeline(sampleRate int) *Pipeline {
	fft := NewFFT(config.FFTSize)
	binCfg := DefaultBinConfig(sampleRate)
	binEdges := LogBinEdges(binCfg)
	agc := NewAGCState(config.NumBins)

	return &Pipeline{
		fft:        fft,
		binEdges:   binEdges,
		agc:        agc,
		sampleRate: sampleRate,
	}
}

// Run starts the DSP processing loop.
// samples: input channel receiving PCM buffers (float32)
// frames: output channel sending processed Frame structs
// The function blocks until context is cancelled.
func (p *Pipeline) Run(ctx context.Context, samples <-chan []float32, frames chan<- Frame) {
	// Buffer for accumulating samples to reach FFTSize
	buffer := make([]float32, 0, config.FFTSize)

	ticker := time.NewTicker(config.TickAnalyzeDuration())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(frames)
			return

		case buf, ok := <-samples:
			if !ok {
				close(frames)
				return
			}

			// Accumulate samples
			buffer = append(buffer, buf...)

			// Process when we have enough samples
			for len(buffer) >= config.FFTSize {
				// Take FFTSize samples
				frameSamples := buffer[:config.FFTSize]
				buffer = buffer[config.FFTSize:]

				// Process this frame
				frame := p.processFrame(frameSamples)
				select {
				case frames <- frame:
				case <-ctx.Done():
					close(frames)
					return
				}
			}

		case <-ticker.C:
			// Periodic analysis tick - if we have partial buffer, process it
			if len(buffer) > 0 {
				// Pad with zeros if needed
				if len(buffer) < config.FFTSize {
					padded := make([]float32, config.FFTSize)
					copy(padded, buffer)
					frame := p.processFrame(padded)
					select {
					case frames <- frame:
					case <-ctx.Done():
						close(frames)
						return
					}
				} else {
					frame := p.processFrame(buffer[:config.FFTSize])
					buffer = buffer[config.FFTSize:]
					select {
					case frames <- frame:
					case <-ctx.Done():
						close(frames)
						return
					}
				}
			}
		}
	}
}

// processFrame runs the full DSP chain on one FFT-size buffer
func (p *Pipeline) processFrame(samples []float32) Frame {
	// 1. Hann window + FFT -> magnitude spectrum
	magnitude := MagnitudeSpectrum(samples, p.fft)

	// 2. Log-frequency binning
	binned := BinMagnitudes(magnitude, p.binEdges)

	// 3. dB conversion + AGC scaling -> 0.0-1.0
	scaled := p.agc.Process(binned)

	return NewFrame(scaled, p.sampleRate)
}