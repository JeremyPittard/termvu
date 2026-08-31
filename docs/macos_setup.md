# macOS Setup

## Audio Capture (BlackHole + Multi-Output Device)

**Phase 1 Status: Documentation only - no runtime implementation**

macOS does not provide a built-in loopback capture API like WASAPI (Windows) or PulseAudio monitor (Linux). To capture system audio output, you must create a virtual audio device setup.

### Required Software

1. **BlackHole** (free, open-source virtual audio driver)
   - Download: https://github.com/ExistentialAudio/BlackHole/releases
   - Install: `brew install --cask blackhole-2ch` (or 16ch for more channels)

2. **Audio MIDI Setup** (built-in macOS utility)

### Setup Steps

#### 1. Install BlackHole
```bash
# Via Homebrew
brew install --cask blackhole-2ch

# Or download installer from GitHub releases
```

#### 2. Create Multi-Output Device
1. Open **Audio MIDI Setup** (Applications → Utilities → Audio MIDI Setup)
2. Click **+** → **Create Multi-Output Device**
3. In the right panel, check:
   - **BlackHole 2ch** (or 16ch)
   - Your **actual output device** (e.g., "MacBook Pro Speakers", "External Headphones", etc.)
4. **Important**: Enable **Drift Correction** for your actual output device (not BlackHole)
5. Right-click the new Multi-Output Device → **Use This Device For Sound Output**

#### 3. Verify
- Play audio - you should hear it normally
- BlackHole now receives the same audio stream
- termvu (Phase 2+) will capture from BlackHole as input device

### Why This Works

```
┌─────────────┐     ┌─────────────────────┐
│  Apps       │────▶│  Multi-Output Dev   │
│ (Chrome,    │     │  ┌───────────────┐  │
│  Spotify,   │     │  │ BlackHole     │  │◀─── termvu captures here
│  Tidal, etc)│     │  │ (capture)     │  │
│             │     │  └───────────────┘  │
└─────────────┘     │  ┌───────────────┐  │
                    │  │ Real Output   │  │◀─── You hear this
                    │  │ (speakers)    │  │
                    │  └───────────────┘  │
                    └─────────────────────┘
```

The Multi-Output Device duplicates audio to both BlackHole (for capture) and your real output (for listening).

## Metadata (NowPlaying)

**Phase 1 Status: Stub implementation only**

macOS provides `NowPlaying` / `MPNowPlayingInfoCenter` for media metadata, but it requires:
- App sandbox entitlements (for App Store distribution)
- Or direct access via AppleScript/osascript (for CLI tools)

### Current Behavior (Phase 1)
- Metadata returns empty
- UI displays "Live System Audio" / "Unknown Track" fallback

### Phase 2 Options
1. **AppleScript/osascript** - Query `Music`, `Spotify`, `Chrome` etc. via scripting
2. **MediaRemote framework** - Private API, not recommended
3. **NowPlaying via Swift/ObjC bridge** - Requires CGO and entitlements

## Build Requirements

- Go 1.27+
- CGO enabled (`CGO_ENABLED=1`) - for miniaudio
- Xcode Command Line Tools: `xcode-select --install`

```bash
# Native build
go build -o termvu ./cmd/termvu
```

## Troubleshooting

### BlackHole not appearing
- Reinstall BlackHole: `brew reinstall --cask blackhole-2ch`
- Restart after install (kernel extension)

### No audio from Multi-Output Device
- Check Drift Correction is enabled for real output, not BlackHole
- Ensure Multi-Output Device is set as system output
- Check volume levels in Audio MIDI Setup

### termvu captures silence (Phase 2+)
- Verify BlackHole is selected as input in termvu
- Check BlackHole input level in Audio MIDI Setup