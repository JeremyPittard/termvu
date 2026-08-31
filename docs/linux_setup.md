# Linux Setup

## Audio Capture (PulseAudio/PipeWire Monitor)

termvu uses `malgo` (miniaudio) to capture from the PulseAudio/PipeWire **monitor** device.

### Requirements
- PulseAudio or PipeWire running
- User in the `audio` group (or appropriate permissions)
- A monitor source available for the default output sink

### Verify Setup

```bash
# Check PulseAudio/PipeWire is running
pactl info

# List monitor sources
pactl list short sources | grep monitor

# Example output:
# 0	alsa_output.pci-0000_00_1f.3.analog-stereo.monitor	module-alsa-card.c	s16le 2ch 48000Hz	IDLE
# 1	alsa_output.usb-... .monitor	module-alsa-card.c	s16le 2ch 48000Hz	IDLE
```

The default monitor (usually index 0 or the one matching your default sink) will be used automatically.

### PipeWire Specifics

PipeWire exposes PulseAudio-compatible monitor devices. If using PipeWire with WirePlumber:
- Monitor devices should appear automatically
- No additional configuration needed

### Permissions

If capture fails with permission errors:
```bash
# Add user to audio group
sudo usermod -a -G audio $USER
# Log out and back in
```

Or run with appropriate capabilities (not recommended for daily use):
```bash
sudo setcap cap_sys_resource+ep $(which termvu)
```

## Metadata (MPRIS)

termvu uses `godbus/dbus` to monitor MPRIS-compatible players via the session bus.

### Supported Players
- Spotify (native or flatpak)
- Chrome/Chromium/Edge/Firefox (with MPRIS extension or native support)
- Firefox (with `media.hardwaremediakeysenabled` = true)
- Tidal (native or web with MPRIS bridge)
- Any MPRIS2-compliant player

### Verify MPRIS

```bash
# List MPRIS players on session bus
dbus-send --session --dest=org.freedesktop.DBus \
  --print-reply /org/freedesktop/DBus \
  org.freedesktop.DBus.ListNames | grep mpris

# Check metadata for a specific player
dbus-send --session --dest=org.mpris.MediaPlayer2.spotify \
  --print-reply /org/mpris/MediaPlayer2 \
  org.freedesktop.DBus.Properties.Get \
  string:org.mpris.MediaPlayer2.Player string:Metadata
```

### Flatpak Spotify

If using Flatpak Spotify, the D-Bus name is typically:
- `org.mpris.MediaPlayer2.spotify` (may need `--filesystem=xdg-run/dbus` permission)

## Build Requirements

- Go 1.27+
- CGO enabled (`CGO_ENABLED=1`) - for miniaudio
- ALSA/PulseAudio development headers:
  ```bash
  # Debian/Ubuntu
  sudo apt install libasound2-dev libpulse-dev

  # Arch
  sudo pacman -S alsa-lib libpulse

  # Fedora
  sudo dnf install alsa-lib-devel pulseaudio-libs-devel
  ```

## Troubleshooting

### No monitor device found
```bash
# Check default sink
pactl get-default-sink

# Check if monitor exists for that sink
pactl list short sources | grep $(pactl get-default-sink | sed 's/\.monitor$//')
```

### MPRIS not detecting players
- Ensure player is running and playing
- Check `dbus-monitor --session` for `NameOwnerChanged` signals
- Some players only expose MPRIS when playing

### Audio stuttering
- Increase `PeriodSizeInMilliseconds` in config (try 15-20ms)
- Check CPU usage - DSP pipeline should be lightweight