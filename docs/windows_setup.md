# Windows Setup

## Audio Capture (WASAPI Loopback)

termvu uses `malgo` (miniaudio) for WASAPI loopback capture on Windows. This works out of the box on Windows 10/11 - no additional software required.

The loopback device captures the **system audio output** (what you hear from speakers/headphones), including:
- Chrome/Edge/Firefox (YouTube, Tidal web, etc.)
- Spotify desktop app
- Tidal desktop app
- Any other application playing audio

## Metadata (GSMTC)

**Phase 1 Status: Stub implementation only**

Windows metadata uses the Global System Media Transport Controls (GSMTC) API via WinRT/COM. There is currently no mature Go binding for this.

### Current Behavior (Phase 1)
- Metadata returns empty
- UI displays "Live System Audio" / "Unknown Track" fallback
- No crash or error - graceful degradation

### Phase 2 Plan
Implement cgo COM interop against:
- `IGlobalSystemMediaTransportControlsSessionManager`
- `IGlobalSystemMediaTransportControlsSession`
- `MediaProperties` for title/artist/album

Alternative: Use a Go wrapper like `github.com/robmccormack/go-gsmtc` if it matures.

## Build Requirements

- Go 1.27+
- CGO enabled (`CGO_ENABLED=1`)
- MinGW-w64 (for miniaudio CGO compilation)

```bash
# Cross-compile from Linux
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -o termvu.exe ./cmd/termvu

# Native build on Windows
go build -o termvu.exe ./cmd/termvu
```

## Troubleshooting

### No audio captured
- Ensure audio is playing on the default output device
- Check Windows Sound settings → Output device matches what you're listening to
- Some apps may use exclusive mode - try disabling "Allow applications to take exclusive control" in Sound Control Panel

### Build fails with CGO errors
- Install MinGW-w64: `pacman -S mingw-w64-x86_64-gcc` (MSYS2) or use TDM-GCC
- Ensure `CC` environment variable points to the correct compiler