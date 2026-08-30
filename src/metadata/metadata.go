package metadata

import (
	"context"
	"time"
)

// Metadata holds currently-playing track information.
type Metadata struct {
	Title   string
	Artist  string
	Album   string
	Playing bool
}

// PollMetadata periodically calls GetMetadata and sends the result on the
// returned channel. It stops when the context is canceled.
func PollMetadata(ctx context.Context, interval time.Duration) <-chan Metadata {
	ch := make(chan Metadata)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			meta, err := GetMetadata()
			if err != nil {
				// Avoid a hot loop when no session is available; the caller's
				// interval still throttles work, so just skip this frame.
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
				continue
			}
			select {
			case <-ctx.Done():
				return
			case ch <- meta:
			case <-ticker.C:
			}
		}
	}()
	return ch
}