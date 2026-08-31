package dsp

import "time"

// Frame represents a single processed spectrum frame
type Frame struct {
	Values     []float64
	Timestamp  time.Time
	SampleRate int
}

// NewFrame creates a new frame with the given values
func NewFrame(values []float64, sampleRate int) Frame {
	return Frame{
		Values:     values,
		Timestamp:  time.Now(),
		SampleRate: sampleRate,
	}
}

// IsEmpty returns true if the frame has no values
func (f Frame) IsEmpty() bool {
	return len(f.Values) == 0
}