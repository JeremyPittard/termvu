//go:build !windows

package metadata

// GetMetadata returns dummy track metadata on non-Windows platforms where the
// Windows.Media.Control (SMTC) API is not available.
func GetMetadata() (Metadata, error) {
	return Metadata{
		Title:   "No track",
		Artist:  "No artist",
		Album:   "No album",
		Playing: false,
	}, nil
}