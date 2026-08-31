//go:build darwin

package media

func newPlatformMonitor() (MonitorInterface, error) {
	return NewNowPlayingMonitor()
}