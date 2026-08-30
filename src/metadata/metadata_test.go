package metadata

import (
	"context"
	"testing"
	"time"
)

func TestGetMetadata(t *testing.T) {
	meta, err := GetMetadata()
	if err != nil {
		t.Fatalf("GetMetadata returned error: %v", err)
	}
	if meta.Title == "" {
		t.Errorf("GetMetadata returned empty title")
	}
}

func TestPollMetadata(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	ch := PollMetadata(ctx, 50*time.Millisecond)
	select {
	case meta, ok := <-ch:
		if !ok {
			t.Error("channel closed unexpectedly")
		}
		if meta.Title == "" {
			t.Errorf("received empty title")
		}
	case <-ctx.Done():
		t.Error("timed out waiting for metadata")
	}
}
