package state

import (
	"sync"
	"time"

	"termvu/media"
)

// MetadataStore provides thread-safe access to the latest metadata
type MetadataStore struct {
	mu        sync.Mutex
	current   media.Metadata
	updatedAt time.Time
}

// NewMetadataStore creates a new metadata store
func NewMetadataStore() *MetadataStore {
	return &MetadataStore{}
}

// Write stores new metadata
func (s *MetadataStore) Write(meta media.Metadata) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = meta
	s.updatedAt = time.Now()
}

// Read returns the current metadata (zero value if never written)
func (s *MetadataStore) Read() media.Metadata {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}

// UpdatedAt returns the time of the last write
func (s *MetadataStore) UpdatedAt() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updatedAt
}

// HasData returns true if metadata has been written
func (s *MetadataStore) HasData() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.current.IsEmpty()
}