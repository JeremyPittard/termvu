//go:build windows

package media

func newPlatformMonitor() (MonitorInterface, error) {
	return NewGSMTCMonitor()
}