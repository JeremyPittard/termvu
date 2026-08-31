//go:build linux

package media

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

// MPRISMonitor watches MPRIS players on the session bus
type MPRISMonitor struct {
	conn       *dbus.Conn
	mu         sync.RWMutex
	players    map[string]*playerInfo
	updateCh   chan Metadata
	stopCh     chan struct{}
	wg         sync.WaitGroup
	lastActive string
	started    bool
}

type playerInfo struct {
	name       string
	owner      string
	path       dbus.ObjectPath
	playing    bool
	metadata   Metadata
	lastUpdate time.Time
}

// NewMPRISMonitor creates a new MPRIS monitor
func NewMPRISMonitor() (*MPRISMonitor, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect to session bus: %w", err)
	}

	m := &MPRISMonitor{
		conn:     conn,
		players:  make(map[string]*playerInfo),
		updateCh: make(chan Metadata, 4),
		stopCh:   make(chan struct{}),
	}

	return m, nil
}

// Start begins monitoring
func (m *MPRISMonitor) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return nil
	}

	// Watch for NameOwnerChanged signals
	err := m.conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.DBus"),
		dbus.WithMatchMember("NameOwnerChanged"),
		dbus.WithMatchObjectPath(dbus.ObjectPath("/org/freedesktop/DBus")),
		dbus.WithMatchSender("org.freedesktop.DBus"),
	)
	if err != nil {
		return fmt.Errorf("add match signal: %w", err)
	}

	// Start signal handler
	m.wg.Add(1)
	go m.signalLoop()

	// Initial scan for existing players
	m.scanPlayers()

	m.started = true
	return nil
}

// UpdateChannel returns the channel that receives metadata updates
func (m *MPRISMonitor) UpdateChannel() <-chan Metadata {
	return m.updateCh
}

// Stop stops the monitor
func (m *MPRISMonitor) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return nil
	}

	close(m.stopCh)
	m.wg.Wait()

	m.conn.RemoveMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.DBus"),
		dbus.WithMatchMember("NameOwnerChanged"),
		dbus.WithMatchObjectPath(dbus.ObjectPath("/org/freedesktop/DBus")),
		dbus.WithMatchSender("org.freedesktop.DBus"),
	)
	return m.conn.Close()
}

// Poll queries all known players (called periodically)
func (m *MPRISMonitor) Poll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, player := range m.players {
		m.queryPlayerLocked(player)
	}
	m.updateLastActive()
}

// signalLoop handles DBus signals
func (m *MPRISMonitor) signalLoop() {
	defer m.wg.Done()

	sigCh := make(chan *dbus.Signal, 16)
	m.conn.Signal(sigCh)

	for {
		select {
		case <-m.stopCh:
			return
		case sig := <-sigCh:
			if sig.Name == "org.freedesktop.DBus.NameOwnerChanged" {
				m.handleNameOwnerChanged(sig)
			}
		}
	}
}

// handleNameOwnerChanged processes NameOwnerChanged signals
func (m *MPRISMonitor) handleNameOwnerChanged(sig *dbus.Signal) {
	if len(sig.Body) < 3 {
		return
	}

	name, _ := sig.Body[0].(string)
	_, _ = sig.Body[1].(string) // oldOwner
	newOwner, _ := sig.Body[2].(string)

	// Only care about MPRIS player names
	if !strings.HasPrefix(name, "org.mpris.MediaPlayer2.") {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if newOwner == "" {
		// Player disconnected
		delete(m.players, name)
		if m.lastActive == name {
			m.updateLastActive()
		}
	} else {
		// Player connected or changed owner
		player := m.players[name]
		if player == nil {
			player = &playerInfo{name: name}
			m.players[name] = player
		}
		player.owner = newOwner
		player.path = dbus.ObjectPath("/org/mpris/MediaPlayer2")
		// Query metadata immediately
		m.queryPlayerLocked(player)
	}
}

// scanPlayers scans for existing MPRIS players on startup
func (m *MPRISMonitor) scanPlayers() {
	// Use ListNames on the DBus bus object
	obj := m.conn.Object("org.freedesktop.DBus", dbus.ObjectPath("/org/freedesktop/DBus"))
	var names []string
	err := obj.Call("org.freedesktop.DBus.ListNames", 0).Store(&names)
	if err != nil {
		return
	}

	for _, name := range names {
		if strings.HasPrefix(name, "org.mpris.MediaPlayer2.") {
			player := &playerInfo{name: name}
			m.players[name] = player
			m.queryPlayerLocked(player)
		}
	}
	m.updateLastActive()
}

// queryPlayerLocked queries a player's metadata and playback status (must hold lock)
func (m *MPRISMonitor) queryPlayerLocked(p *playerInfo) {
	if p.owner == "" {
		return
	}

	obj := m.conn.Object(p.owner, p.path)

	// Get PlaybackStatus
	var status string
	err := obj.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.mpris.MediaPlayer2.Player", "PlaybackStatus").Store(&status)
	if err != nil {
		return
	}
	p.playing = (status == "Playing")

	// Get Metadata
	var metadata map[string]dbus.Variant
	err = obj.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.mpris.MediaPlayer2.Player", "Metadata").Store(&metadata)
	if err != nil {
		return
	}

	p.metadata = parseMetadata(metadata, p.name)
	p.lastUpdate = time.Now()

	// Send update if this is the active player
	if m.lastActive == p.name || (p.playing && m.lastActive == "") {
		select {
		case m.updateCh <- p.metadata:
		default:
		}
	}
}

// updateLastActive updates the last active player (prefers Playing)
func (m *MPRISMonitor) updateLastActive() {
	var best *playerInfo
	for _, p := range m.players {
		if p.playing {
			best = p
			break
		}
		if best == nil || p.lastUpdate.After(best.lastUpdate) {
			best = p
		}
	}
	if best != nil {
		m.lastActive = best.name
	}
}

// parseMetadata extracts track info from MPRIS metadata map
func parseMetadata(raw map[string]dbus.Variant, playerName string) Metadata {
	m := Metadata{PlayerName: playerName}

	if v, ok := raw["xesam:title"]; ok {
		if s, ok := v.Value().(string); ok {
			m.Title = s
		}
	}
	if v, ok := raw["xesam:artist"]; ok {
		// Artist is an array of strings
		if arr, ok := v.Value().([]string); ok && len(arr) > 0 {
			m.Artist = arr[0]
		}
	}
	if v, ok := raw["xesam:album"]; ok {
		if s, ok := v.Value().(string); ok {
			m.Album = s
		}
	}

	m.Timestamp = time.Now()
	return m
}