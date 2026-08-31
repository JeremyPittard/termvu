package state

import (
	"testing"
	"time"

	"termvu/dsp"
	"termvu/media"
)

func TestFrameBuffer(t *testing.T) {
	buf := NewFrameBuffer()

	// Read before write should return zero frame
	frame := buf.Read()
	if !frame.IsEmpty() {
		t.Error("Expected empty frame before write")
	}
	if buf.HasData() {
		t.Error("Expected HasData() = false before write")
	}

	// Write a frame
	testFrame := dsp.NewFrame([]float64{0.1, 0.2, 0.3}, 48000)
	buf.Write(testFrame)

	// Read should return the frame
	frame = buf.Read()
	if frame.IsEmpty() {
		t.Error("Expected non-empty frame after write")
	}
	if len(frame.Values) != 3 {
		t.Errorf("Expected 3 values, got %d", len(frame.Values))
	}
	if frame.Values[0] != 0.1 || frame.Values[1] != 0.2 || frame.Values[2] != 0.3 {
		t.Errorf("Frame values mismatch: %v", frame.Values)
	}
	if frame.SampleRate != 48000 {
		t.Errorf("Expected sample rate 48000, got %d", frame.SampleRate)
	}
	if !buf.HasData() {
		t.Error("Expected HasData() = true after write")
	}

	// UpdatedAt should be recent
	if time.Since(buf.UpdatedAt()) > time.Second {
		t.Error("UpdatedAt should be recent")
	}
}

func TestMetadataStore(t *testing.T) {
	store := NewMetadataStore()

	// Read before write should return empty metadata
	meta := store.Read()
	if !meta.IsEmpty() {
		t.Error("Expected empty metadata before write")
	}
	if store.HasData() {
		t.Error("Expected HasData() = false before write")
	}

	// Write metadata
	testMeta := media.Metadata{
		Title:      "Test Title",
		Artist:     "Test Artist",
		Album:      "Test Album",
		PlayerName: "Test Player",
	}
	store.Write(testMeta)

	// Read should return the metadata
	meta = store.Read()
	if meta.IsEmpty() {
		t.Error("Expected non-empty metadata after write")
	}
	if meta.Title != "Test Title" {
		t.Errorf("Expected title 'Test Title', got '%s'", meta.Title)
	}
	if meta.Artist != "Test Artist" {
		t.Errorf("Expected artist 'Test Artist', got '%s'", meta.Artist)
	}
	if meta.Album != "Test Album" {
		t.Errorf("Expected album 'Test Album', got '%s'", meta.Album)
	}
	if meta.PlayerName != "Test Player" {
		t.Errorf("Expected player 'Test Player', got '%s'", meta.PlayerName)
	}
	if !store.HasData() {
		t.Error("Expected HasData() = true after write")
	}

	// UpdatedAt should be recent
	if time.Since(store.UpdatedAt()) > time.Second {
		t.Error("UpdatedAt should be recent")
	}
}

func TestFrameBufferConcurrency(t *testing.T) {
	buf := NewFrameBuffer()
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 1000; i++ {
			frame := dsp.NewFrame([]float64{float64(i) * 0.001}, 48000)
			buf.Write(frame)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 1000; i++ {
			_ = buf.Read()
		}
		done <- true
	}()

	// Wait for both
	<-done
	<-done

	// Should not panic or race
	if !buf.HasData() {
		t.Error("Expected data after concurrent writes")
	}
}

func TestMetadataStoreConcurrency(t *testing.T) {
	store := NewMetadataStore()
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 1000; i++ {
			meta := media.Metadata{
				Title:  "Title",
				Artist: "Artist",
			}
			store.Write(meta)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 1000; i++ {
			_ = store.Read()
		}
		done <- true
	}()

	<-done
	<-done

	if !store.HasData() {
		t.Error("Expected data after concurrent writes")
	}
}