package audio

import (
	"context"
	"testing"
	"time"
)

func TestAudioCapture_Start(t *testing.T) {
	ac, err := NewAudioCapture(44100, 1024)
	if err != nil {
		t.Fatalf("failed to create audio capture: %v", err)
	}
	defer ac.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	ch := ac.Start(ctx)
	select {
	case spec, ok := <-ch:
		if !ok {
			t.Error("channel closed unexpectedly")
		}
		// Check that we got a spectrum with 30 elements.
		if len(spec) != 30 {
			t.Errorf("expected spectrum length 30, got %d", len(spec))
		}
		// Check that values are between 0 and 1 (or at least not negative).
		for i, v := range spec {
			if v < 0 {
				t.Errorf("spectrum[%d] = %f, expected >= 0", i, v)
			}
		}
	case <-ctx.Done():
		t.Error("timed out waiting for spectrum")
	}
}
