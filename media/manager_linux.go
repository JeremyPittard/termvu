//go:build linux

package media

func newPlatformMonitor() (MonitorInterface, error) {
	return NewMPRISMonitor()
}