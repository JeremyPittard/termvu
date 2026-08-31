package config

// Build tags documentation:
// - windows: Windows-specific code (GSMTC cgo COM interop)
// - linux: Linux-specific code (MPRIS via godbus/dbus)
// - darwin: macOS-specific code (NowPlaying stub, BlackHole docs)
//
// CGO requirements:
// - windows: required for GSMTC (if implemented beyond stub)
// - linux: no CGO (godbus/dbus is pure Go)
// - darwin: no CGO (stub only)
//
// malgo uses CGO internally (miniaudio) but builds on all platforms.
// Cross-compile: GOOS=windows GOARCH=amd64 go build works on Linux host.
const BuildTagsDoc = `
Build Tags:
  windows  - Windows GSMTC metadata (cgo COM)
  linux    - Linux MPRIS metadata (pure Go)
  darwin   - macOS NowPlaying stub + BlackHole docs

CGO:
  Required on Windows for GSMTC implementation.
  Not required on Linux (godbus/dbus pure Go).
  Not required on macOS (stub only).
`