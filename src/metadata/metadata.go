package metadata

import (
	"context"
	"time"
)

// Metadata holds track information.
type Metadata struct {
	Title  string
	Artist string
	Album  string
	Playing bool
}

// GetMetadata returns the current track metadata from Windows NowPlaying.
// This is a stub implementation that returns dummy data.
func GetMetadata() (Metadata, error) {
	return Metadata{
		Title:  "Dummy Title",
		Artist: "Dummy Artist",
		Album:  "Dummy Album",
		Playing: true,
	}, nil
}

// PollMetadata periodically calls GetMetadata and sends results on the returned channel.
// It stops when the context is canceled.
func PollMetadata(ctx context.Context, interval time.Duration) <-chan Metadata {
	ch := make(chan Metadata)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			meta, err := GetMetadata()
			if err != nil {
				// TODO: handle error
				continue
			}
			select {
			case <-ctx.Done():
				return
			case ch <- meta:
			}
			<-ticker.C
		}
	}()
	return ch
}
