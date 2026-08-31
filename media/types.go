package media

import "time"

// Metadata holds media track information
type Metadata struct {
	Title      string
	Artist     string
	Album      string
	PlayerName string
	Timestamp  time.Time
}

// IsEmpty returns true if no metadata is set
func (m Metadata) IsEmpty() bool {
	return m.Title == "" && m.Artist == "" && m.Album == ""
}

// DisplayTitle returns a display-friendly title string
func (m Metadata) DisplayTitle() string {
	if m.Title != "" {
		if m.Artist != "" {
			return m.Title + " — " + m.Artist
		}
		return m.Title
	}
	return "Live System Audio"
}

// DisplaySubtitle returns a display-friendly subtitle string
func (m Metadata) DisplaySubtitle() string {
	if m.Album != "" {
		return m.Album
	}
	if m.PlayerName != "" {
		return "via " + m.PlayerName
	}
	return "Unknown Track"
}