---
title: "System-Wide Terminal Audio Visualizer - Plan"
type: feat
date: 2026-08-31
origin: "ce-plan-bootstrap"
artifact_contract: ce-unified-plan/v1
artifact_readiness: implementation-ready
execution: code
---

## Goal Capsule

- **Objective:** Build a cross-platform Go TUI that captures live system desktop audio output (Tidal, YouTube, Chrome, Spotify), processes it via FFT into a responsive spectrum visualizer, extracts active track metadata via OS media hooks, and renders in the terminal at 30–60 FPS using bubbletea v2 + lipgloss v2.
- **Authority hierarchy:** This plan > cliamp architectural reference > implementation judgment.
- **Execution profile:** Two strict phases — Phase 1 proves the live data pipeline with zero custom animation (mirror cliamp's tick loop and state struct); Phase 2 is developer-driven polish (attack/decay, peak-hold gravity, block chars, color gradients).
- **Stop conditions:** Phase 1 complete when real audio in Chrome/Tidal drives visible block height changes + active track info displays, with no stutter, freeze, or audio-blocking.
- **Tail ownership:** ce-work executes Phase 1 units in dependency order; Phase 2 is manual tuning post-Phase-1.

## Product Contract

### Summary

A system-wide terminal audio visualizer that hooks the OS audio output loopback (WASAPI on Windows, Pulse/PipeWire monitor on Linux, BlackHole doc-only on macOS), runs a real-time DSP pipeline (Hann window → FFT → log-frequency binning → dB + AGC scaling) in a background goroutine, reads media metadata via platform hooks (GSMTC Windows, MPRIS Linux, NowPlaying macOS), and renders a block-character spectrum + metadata header via bubbletea's alternate-screen renderer at 30–60 FPS.

### Problem Frame

Users want a terminal-native "now playing" visualizer that works with *any* desktop audio source — not just local files, not just one player. Existing tools either decode local files (cliamp, cava) or require per-app integration. The architectural pivot from cliamp is replacing the file/stream decoder with a real-time OS loopback capture engine while keeping cliamp's proven windowing, log-binning, state, and render loop. The hard constraint: never write modified audio back to speakers; all EQ/gain is visual-only math.

### Requirements

- **R1.** Capture raw 16-bit or 32-bit float PCM from the default desktop audio output on Windows (WASAPI Loopback via malgo/miniaudio) and Linux (PulseAudio/PipeWire monitor via malgo). Sub-15 ms processing latency (input buffer 512–1024 samples).
- **R2.** Run audio capture + DSP (Hann window → Real FFT → log-frequency bin aggregation → dB conversion + AGC fallback) in a dedicated background goroutine, never on the TUI thread.
- **R3.** Aggregate FFT energy magnitudes into N visual columns mapped logarithmically across ~30 Hz–16 kHz. Output per-frame float array (0.0–1.0) to a thread-safe shared buffer.
- **R4.** Display-only visual EQ/scaling: convert linear amplitudes to dB, apply AGC fallback, optional per-band display multipliers. Strictly visual — no audio output modification.
- **R5.** Extract active media metadata (Title, Artist, Album) concurrently: Windows via GSMTC (GlobalSystemMediaTransportControlsSessionManager), Linux via MPRIS (DBus `org.mpris.MediaPlayer2.*`), macOS via NowPlaying/MPNowPlayingInfoCenter or AppleScript (documented setup only in Phase 1).
- **R6.** Graceful metadata fallback: if no OS hook yields metadata, render "Live System Audio" / "Unknown Track" label without crashing.
- **R7.** Thread-safe metadata store; async metadata poll every 1–2 s via `tea.Every`, not every render tick.
- **R8.** Drive bubbletea render loop via `tea.Every` at 30–60 FPS (16–33 ms). On each tick, read latest frame from shared atomic buffer and render.
- **R9.** Render via bubbletea v2 `tea.NewView` with `AltScreen = true` — zero manual ANSI, zero `fmt.Print` per frame. This prevents the "repeatedly prints lines" failure mode.
- **R10.** Phase 1: zero custom animation — mirror cliamp's frame update logic and basic state struct. Copy cliamp's tick cadence (16 ms anim, 33 ms FFT analysis) and atomic state pattern.

### Scope Boundaries

#### Deferred for later

- macOS loopback capture implementation (Phase 1 is documented setup only: BlackHole + Multi-Output Device)
- Custom spring physics, complex decay curves, smooth interpolation math (Phase 2)
- Per-band display EQ UI controls (Phase 2)
- Multiple visualizer modes (bars, wave, scope, terrain, etc.) — Phase 1 ships one spectrum mode
- Plugin/extension system
- Config file hot-reload

#### Outside this product's identity

- Audio playback, decoding, or streaming — this is a *visualizer only*
- Writing audio back to system output (strictly forbidden by R4)
- Per-app volume control or media control (play/pause/next) — metadata is read-only

### Outstanding Questions

- **Q1 (blocking):** No mature Go GSMTC binding exists. Windows metadata will need cgo COM interop or a stub with graceful fallback. Resolve approach before U4.
- **Q2 (deferred):** Exact log-frequency bin mapping (N columns, min/max Hz, bin edges) — tunable constant, defer to Phase 1 verification.
- **Q3 (deferred):** AGC algorithm parameters (attack/release time constants, target dBFS) — Phase 2 tuning.

### Sources

- cliamp (github.com/bjarneo/cliamp) — architecture reference for tick loop, FFT, state, render
- malgo v0.11.26 (gen2brain/malgo) — cross-platform audio capture (WASAPI Loopback, Pulse, PipeWire, CoreAudio)
- go-dsp v1.0.0 (mjibson/go-dsp) — FFT alternative; cliamp uses hand-rolled radix-2 with twiddle table
- godbus/dbus v5.2.2 — Linux MPRIS monitoring (inverse of cliamp's MPRIS server)
- bubbletea v2.0.9 + lipgloss v2.0.6 — TUI framework (v2 API: `tea.NewView`, `tea.Tick(d, fn)`)

## Planning Contract

### Key Technical Decisions

- **KTD1: Audio backend — malgo/miniaudio as primary, portaudio documented alternative.** malgo exposes native `Loopback` device type, `BackendWasapi`, `PeriodSizeInMilliseconds` mapping directly to the 512–1024 sample / sub-15ms constraint. Portaudio's loopback support is weaker/OS-specific. An internal `CaptureBackend` interface keeps alternatives swappable.
- **KTD2: DSP FFT — use cliamp's hand-rolled radix-2 Cooley-Tukey with precomputed twiddle table (ui/fft.go) rather than go-dsp.** cliamp's implementation is proven, zero-allocation in the hot path, and matches the log-binning approach. go-dsp is retained as a documented alternative.
- **KTD3: Thread communication — mutex-protected struct (not atomics) for the shared frame buffer.** Frame size is small (N floats + metadata), contention is low (one writer goroutine, one reader TUI thread). Simpler than lock-free; matches cliamp's pattern.
- **KTD4: Metadata polling — `tea.Every(1500*time.Millisecond, func() tea.Msg { return metadataTickMsg{} })`** independent of render tick. Linux MPRIS uses `godbus/dbus` to watch `NameOwnerChanged` + poll `Metadata` property; Windows GSMTC uses cgo COM (or stub); macOS documented-only.
- **KTD5: Render loop — bubbletea v2 `tea.NewProgram(model, tea.WithAltScreen())` with `View() tea.View` returning `tea.NewView(rendered)` and `AltScreen = true`.** No `fmt.Print`, no manual cursor/clear. This is the hard fix for "repeatedly prints lines."
- **KTD6: Tick cadence — mirror cliamp exactly: `TickAnim = 16ms` (60 FPS render), `TickAnalyze = 33ms` (30 Hz FFT analysis).** Render tick reads latest frame; analysis tick triggers DSP work. Decoupled cadences prevent DSP from blocking render.
- **KTD7: Windowing — Hann window applied to PCM buffer before FFT.** Matches cliamp's spectral leakage elimination.
- **KTD8: Log-frequency binning — map FFT bins to N visual columns logarithmically across ~30 Hz–16 kHz.** Exact N, min/max Hz, bin edges are tunable constants (Q2).
- **KTD9: Project structure — clean separation: `/audio` (capture backends), `/dsp` (windowing, FFT, binning, AGC), `/media` (metadata hooks per OS), `/ui` (bubbletea model, view, styles), `/config` (constants, tunables).**

### High-Level Technical Design

```mermaid
flowchart TB
    subgraph OS_Audio[OS Audio Loopback]
        WASAPI[Windows: WASAPI Loopback]
        PA[Linux: Pulse/PipeWire Monitor]
        BH[macOS: BlackHole Doc-Only]
    end

    subgraph Capture[Audio Capture Goroutine]
        Backend[malgo CaptureBackend\nInterface]
        Loop[Capture Loop\n512-1024 samples @ PeriodSize]
    end

    subgraph DSP[DSP Pipeline Goroutine]
        Win[Hann Window]
        FFT[Radix-2 FFT\nTwiddle Table]
        Bin[Log-Freq Binning\n30Hz-16kHz → N cols]
        AGC[dB + AGC Scaling\n0.0-1.0 float[]]
    end

    subgraph State[Thread-Safe Shared State]
        Buf[(FrameBuffer\nmutex-protected)]
        Meta[(MetadataStore\nmutex-protected)]
    end

    subgraph Media[Metadata Goroutine]
        GSMTC[Windows: GSMTC\ncgo COM / stub]
        MPRIS[Linux: MPRIS\ngodbus/dbus watch]
        NP[macOS: NowPlaying\nDoc-Only]
    end

    subgraph TUI[Bubbletea v2 Main Thread]
        Tick[tea.Every 16ms\nRender Tick]
        View[View() → tea.NewView\nAltScreen=true]
    end

    OS_Audio --> Backend
    Backend --> Loop
    Loop --> Win
    Win --> FFT
    FFT --> Bin
    Bin --> AGC
    AGC --> Buf
    Media --> Meta
    Tick --> Buf
    Tick --> Meta
    Buf --> View
    Meta --> View
```

**Data flow lifecycle (per PCM frame):**
1. Capture goroutine reads 512–1024 samples from malgo `Device` callback (loopback mode)
2. DSP goroutine: Hann window → radix-2 FFT (precomputed twiddles) → magnitude spectrum
3. Log-frequency binning: aggregate magnitudes into N columns across 30 Hz–16 kHz
4. dB conversion + AGC fallback → normalize to 0.0–1.0 float[N]
5. Lock `FrameBuffer`, write frame + timestamp, unlock
6. TUI `tea.Every(16ms)` tick fires → `View()` reads `FrameBuffer`, renders spectrum + metadata
7. Independent `tea.Every(1500ms)` metadata tick → platform hook → lock `MetadataStore` → update

### Assumptions

- **A1:** malgo's `Loopback` device type + `PeriodSizeInMilliseconds` reliably delivers sub-15ms buffers on Windows (WASAPI) and Linux (Pulse/PipeWire). If not, fallback to `Duplex` with explicit monitor device selection.
- **A2:** Linux MPRIS monitoring via `godbus/dbus` watching `NameOwnerChanged` on session bus + polling `Metadata` property on active `org.mpris.MediaPlayer2.*` players works for Spotify, Chrome, Firefox, Tidal desktop.
- **A3:** Windows GSMTC metadata requires custom cgo COM interop (no mature Go binding exists). A stub returning empty metadata with graceful fallback is acceptable for Phase 1.
- **A4:** cliamp's hand-rolled FFT + twiddle table is performant enough for real-time N=1024 → log-binned ~60 columns at 30 Hz.
- **A5:** bubbletea v2's `tea.NewView` + `AltScreen=true` eliminates the "prints lines" failure mode entirely.
- **A6:** Single spectrum visualizer mode (vertical bars) is sufficient for Phase 1; mode switching is Phase 2.

### Implementation Constraints

- Go 1.27 (local), modules in `termvu/`
- bubbletea v2 + lipgloss v2 (charm.land imports)
- No CGO on Linux/macOS (malgo is pure Go + miniaudio CGO; godbus is pure Go). Windows GSMTC metadata requires CGO — isolated to `/media/windows_gstmc.go` with build tag.
- Input buffer 512–1024 samples, 44.1/48 kHz → ~11–23 ms per buffer at 44.1 kHz; `PeriodSizeInMilliseconds=10` targets ~10 ms
- Thread safety: `sync.Mutex` for both `FrameBuffer` and `MetadataStore`; hold locks only for memcpy, never for I/O

## Implementation Units

### U1. Project scaffolding + go.mod + build tags

**Goal:** Initialize Go module, declare dependencies, set up OS-specific build tags.
**Requirements:** R1, R9
**Dependencies:** (none)
**Files:**
- `go.mod`
- `go.sum`
- `Makefile` (build, test, run targets)
- `config/constants.go` (tunables: sample rate, buffer size, N columns, min/max Hz, tick intervals)
- `config/build_tags.go` (build constraints doc)
**Approach:** `go mod init termvu`. Deps: `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `github.com/gen2brain/malgo`, `github.com/godbus/dbus/v5`, `github.com/mjibson/go-dsp` (alternative). Build tags: `windows` for GSMTC cgo, `linux` for MPRIS, `darwin` for doc-only.
**Patterns:** cliamp `go.mod` structure, `mise.toml` for toolchain pinning.
**Test scenarios:** `Test expectation: none — scaffolding only.`
**Verification:** `go build ./...` succeeds on Linux; cross-compile check for Windows (`GOOS=windows GOARCH=amd64 go build`).

---

### U2. Audio capture backend abstraction + malgo implementation

**Goal:** Define `CaptureBackend` interface and implement malgo backend for Windows (WASAPI Loopback) + Linux (Pulse/PipeWire monitor).
**Requirements:** R1
**Dependencies:** U1
**Files:**
- `audio/backend.go` (interface: `Start(ctx) error`, `Stop() error`, `Samples() <-chan []float32`, `SampleRate() int`, `Channels() int`)
- `audio/malgo_backend.go` (malgo implementation, `Loopback` device type, `PeriodSizeInMilliseconds=10`)
- `audio/malgo_backend_windows.go` (WASAPI-specific config: `NoAutoConvertSRC`, `NoDefaultQualitySRC`)
- `audio/malgo_backend_linux.go` (Pulse monitor device selection: `default.monitor` or PipeWire equivalent)
- `audio/malgo_backend_darwin.go` (stub: returns error "macOS: use BlackHole + Multi-Output Device — see docs/")
**Approach:** malgo `Context` with `BackendWasapi`/`BackendPulse`/`BackendCoreAudio`. `DeviceConfig` with `Capture` subconfig, `DeviceType: Loopback`, `PeriodSizeInMilliseconds: 10`. Callback pushes `[]float32` to channel. Sample rate from device negotiated rate. Channel count from device (stereo → 2, mono → 1, downmix to mono in DSP).
**Patterns:** cliamp `player/tap.go` for callback pattern; malgo examples for device config.
**Test scenarios:**
- Happy: `Start()` opens loopback device, `Samples()` channel delivers buffers at ~100 Hz
- Edge: Device unavailable (no loopback) → `Start()` returns descriptive error
- Error: `Stop()` closes device cleanly, channel closes
- Integration: 5-second capture produces non-zero buffers
**Verification:** `go test ./audio/... -run TestMalgoBackend` (integration, requires audio playing)

---

### U3. DSP pipeline: windowing, FFT, log-binning, dB/AGC

**Goal:** Pure-Go DSP goroutine consuming PCM buffers, producing normalized float[N] frames.
**Requirements:** R2, R3, R4
**Dependencies:** U2
**Files:**
- `dsp/window.go` (Hann window, in-place)
- `dsp/fft.go` (radix-2 Cooley-Tukey + twiddle table, mirrored from cliamp `ui/fft.go`)
- `dsp/binning.go` (log-frequency bin edges, magnitude aggregation → N columns)
- `dsp/scaling.go` (linear→dB, AGC fallback, per-band multipliers, normalize 0.0–1.0)
- `dsp/pipeline.go` (orchestrates: window → FFT → bin → scale; runs in background goroutine)
- `dsp/types.go` (`Frame` struct: `Values []float64`, `Timestamp time.Time`, `SampleRate int`)
**Approach:** `pipeline.Run(ctx, samples <-chan []float32, frames chan<- Frame)` — single goroutine. Twiddle table precomputed at `N=1024` (power of 2). Log bin edges: `minHz=30`, `maxHz=16000`, `numBins=60` (tunable). AGC: track running max, target -6 dBFS, attack/release ~100ms/500ms (tunable, Q3). Per-band multipliers default 1.0.
**Patterns:** cliamp `ui/fft.go` (radix-2 + twiddles), `ui/visualizer.go` binning logic.
**Test scenarios:**
- Happy: 440 Hz sine input → energy concentrated in correct log bin
- Edge: Silent input → all zeros after AGC floor
- Edge: Impulse input → broadband energy across bins
- Error: Non-power-of-2 FFT size → panic with clear message (precondition)
- Integration: White noise → roughly flat log-binned spectrum
**Verification:** `go test ./dsp/... -v` (unit + property tests)

---

### U4. Metadata hooks: Linux MPRIS (godbus/dbus)

**Goal:** Background goroutine monitoring `org.mpris.MediaPlayer2.*` players, emitting metadata updates.
**Requirements:** R5, R6, R7
**Dependencies:** U1
**Files:**
- `media/types.go` (`Metadata` struct: `Title`, `Artist`, `Album`, `PlayerName`, `Timestamp`)
- `media/linux_mpris.go` (DBus session bus, `NameOwnerChanged` watch, `Metadata` property read, `PlaybackStatus` filter)
- `media/manager.go` (unified `MetadataManager` interface, platform dispatch)
**Approach:** `godbus/dbus` connect session bus. `AddMatchSignal` for `NameOwnerChanged` on `org.mpris.MediaPlayer2`. On new owner, introspect `/org/mpris/MediaPlayer2`, read `Metadata` (map[string]variant) + `PlaybackStatus`. Parse `xesam:title`, `xesam:artist` (array), `xesam:album`. Emit `MetadataUpdateMsg` via `tea.Every(1500ms)` poll + signal-driven immediate update. Track last-seen per player; prefer `Playing` status.
**Patterns:** cliamp `mediactl/service_linux.go` (inverse: we *watch* not *export*). `mediactl/metadata.go` types.
**Test scenarios:**
- Happy: Spotify playing → metadata extracted (title, artist, album)
- Edge: Multiple MPRIS players → prefer `Playing` status, fallback to most recent
- Edge: No MPRIS players running → metadata store empty, fallback label used
- Error: DBus connection fails → log error, metadata store stays empty (no crash)
**Verification:** `go test ./media/... -run TestMPRIS` (requires DBus session + Spotify/player running)

---

### U5. Metadata hooks: Windows GSMTC (cgo COM stub + graceful fallback)

**Goal:** Windows metadata via GSMTC; Phase 1 ships stub with fallback, cgo implementation deferred or experimental.
**Requirements:** R5, R6, R7, Q1
**Dependencies:** U1
**Files:**
- `media/windows_gstmc.go` (build tag `windows`; cgo COM interop skeleton OR stub returning empty)
- `media/manager.go` (extend with Windows path)
**Approach:** **Decision required (Q1).** Option A: cgo against `IGlobalSystemMediaTransportControlsSessionManager` — high effort, COM lifetime management. Option B: stub returning zero metadata + log "Windows metadata unavailable — install Go GSMTC binding or use fallback". **Default: Option B for Phase 1** — documented in `docs/windows_setup.md`. GSMTC implementation moves to Phase 2 or follow-up.
**Patterns:** None in cliamp (cliamp is MPRIS server). External ref: Windows `windows.media.control` WinRT API.
**Test scenarios:**
- Happy (stub): Metadata store empty → UI renders "Live System Audio" fallback
- Happy (cgo, if implemented): Active Chrome/Tidal session → metadata extracted
- Edge: No active media session → fallback label
- Error: COM init fails → log, fall back to stub behavior
**Verification:** Cross-compile `GOOS=windows go build`; runtime test on Windows box.

---

### U6. Metadata hooks: macOS (documented setup only)

**Goal:** Document BlackHole + Multi-Output Device setup for macOS loopback; NowPlaying metadata hook stub.
**Requirements:** R5, R6 (doc-only)
**Dependencies:** U1
**Files:**
- `docs/macos_setup.md` (BlackHole install, Audio MIDI Setup Multi-Output Device, select as output)
- `media/darwin_nowplaying.go` (build tag `darwin`; stub returning empty + doc link)
**Approach:** No implementation in Phase 1. Document exact Audio MIDI Setup steps. NowPlaying metadata stub mirrors Windows stub.
**Patterns:** cliamp `mediactl/service_darwin.go` (stub).
**Test scenarios:** `Test expectation: none — documentation only.`
**Verification:** `ls docs/macos_setup.md` exists and is accurate.

---

### U7. Thread-safe shared state: FrameBuffer + MetadataStore

**Goal:** Mutex-protected shared state between capture/DSP goroutines and TUI thread.
**Requirements:** R7, R8
**Dependencies:** U2, U3, U4, U5
**Files:**
- `state/frame_buffer.go` (`FrameBuffer` struct: `mu sync.Mutex`, `Current Frame`, `UpdatedAt time.Time`; `Write(Frame)`, `Read() Frame`)
- `state/metadata_store.go` (`MetadataStore` struct: `mu sync.Mutex`, `Current Metadata`, `UpdatedAt time.Time`; `Write(Metadata)`, `Read() Metadata`)
- `state/types.go` (re-export `Frame`, `Metadata`)
**Approach:** Simple `sync.Mutex` — one writer (DSP or metadata goroutine), one reader (TUI tick). Lock held only for struct copy. `Read()` returns zero-value if never written (triggers fallback).
**Patterns:** cliamp `internal/fileutil/atomic.go` (mutex pattern); `ui/model/tick.go` visualizer tick context.
**Test scenarios:**
- Happy: Concurrent Write/Read — no race (`go test -race`)
- Edge: Read before any Write → zero frame (len 0) + zero metadata → fallback rendering
- Error: Panic in Write → mutex not left locked (defer unlock)
**Verification:** `go test ./state/... -race`

---

### U8. Bubbletea v2 model: tick loop, View(), AltScreen render

**Goal:** Main TUI model with dual tick cadences (render 16ms, metadata poll 1500ms), alternate-screen rendering.
**Requirements:** R8, R9, R10
**Dependencies:** U7
**Files:**
- `ui/model.go` (`Model` struct: `frameBuffer *FrameBuffer`, `metaStore *MetadataStore`, `width/height int`, `visualizer VisState`)
- `ui/tick.go` (tick constants: `TickRender=16ms`, `TickMeta=1500ms`; `tickCmd()` returning `tea.Tick`)
- `ui/update.go` (`Update(msg tea.Msg)` handling `tickMsg`, `metadataTickMsg`, `windowSizeMsg`, `keyMsg` for quit)
- `ui/view.go` (`View() tea.View` → `tea.NewView(rendered)` with `AltScreen=true`, `BackgroundColor`, `ForegroundColor`)
- `ui/styles.go` (lipgloss styles: spectrum colors, metadata header, fallback label)
- `ui/visualizer.go` (spectrum rendering: map float[N] → block chars `█▓▒░` or `▇▆▅▄▃▂▁`, color gradient)
**Approach:** Mirror cliamp `ui/model/tick.go` + `ui/model/view.go` exactly. `tickCmdAt(d)` helper. `View()` recomputes layout, reads `frameBuffer.Read()`, `metaStore.Read()`, builds string via `lipgloss` + block chars, returns `tea.NewView(s).AltScreen(true)`. Quit on `q`/`ctrl+c`.
**Patterns:** cliamp `ui/model/tick.go`, `ui/model/view.go`, `ui/visualizer.go` (spectrum rendering, ANSI caching via lipgloss).
**Test scenarios:**
- Happy: `View()` returns non-empty string with `AltScreen=true`
- Edge: Terminal resize → `windowSizeMsg` updates dimensions, re-renders
- Edge: Zero frame → renders fallback "Live System Audio" + empty spectrum
- Integration: Run program for 5s with mock frame buffer → no "prints lines", stable animation
**Verification:** `go test ./ui/... -v`; manual `go run .` with mock backend shows stable 60 FPS spectrum.

---

### U9. Main entry + wiring + CLI flags

**Goal:** `main.go` wires all goroutines, starts bubbletea program, handles graceful shutdown.
**Requirements:** R1–R10 (integration)
**Dependencies:** U1–U8
**Files:**
- `main.go` (entry: parse flags, init backends, start goroutines, `tea.NewProgram`, `prog.Run()`)
- `cmd/termvu/main.go` (if using cmd/ pattern)
- `docs/windows_setup.md` (GSMTC stub note, malgo WASAPI notes)
- `docs/linux_setup.md` (Pulse/PipeWire monitor device selection)
**Approach:** `main()`: parse `-backend=malgo` (default), `-device=auto`, `-bins=60`, `-fps=60`. Construct `FrameBuffer`, `MetadataStore`. Start capture goroutine → samples chan → DSP goroutine → frames chan → `FrameBuffer.Write`. Start metadata manager goroutine → `MetadataStore.Write`. `tea.NewProgram(model, tea.WithAltScreen(), tea.WithFPS(60))`. `prog.Run()` blocks. On quit: signal ctx cancel, wait for goroutines (bounded).
**Patterns:** cliamp `main.go` program construction + `progOpts`.
**Test scenarios:**
- Happy: Starts, captures audio, renders spectrum, quits cleanly on `q`
- Edge: Audio device busy → prints error, exits non-zero
- Edge: Terminal too small → renders "Terminal too small" message (cliamp pattern)
**Verification:** `go run .` with audio playing → visible spectrum + metadata. Cross-compile Windows.

---

### U10. Phase 1 verification script + CI smoke test

**Goal:** Automated verification that Phase 1 success criteria are met.
**Requirements:** R1–R10 (verification)
**Dependencies:** U9
**Files:**
- `scripts/phase1_verify.sh` (starts app with synthetic audio, checks for stable render, no stderr errors)
- `.github/workflows/ci.yml` (Go build, test, vet, staticcheck; Linux only)
**Approach:** Synthetic audio via `speaker-test` (Linux) or generated sine wave. Run app for 10s, capture stdout/stderr, verify no "prints lines" (no scrolling output), verify spectrum chars present.
**Patterns:** cliamp CI (if any); standard Go CI.
**Test scenarios:**
- CI: `go build`, `go test ./...`, `go vet`, `staticcheck`
- Smoke: App runs 10s without crash, stderr clean
**Verification:** CI passes on push.

---

## Verification Contract

| Gate | Command | Applies To | Success Criteria |
|------|---------|------------|------------------|
| Build | `go build ./...` | All units | Clean build, no vet errors |
| Unit tests | `go test ./audio/... ./dsp/... ./state/... ./media/... ./ui/...` | U2, U3, U4, U7, U8 | All pass, `-race` clean |
| Integration | `go run .` (manual, audio playing) | U9 | Spectrum renders, metadata shows, no line-printing, 60 FPS stable |
| Cross-compile | `GOOS=windows GOARCH=amd64 go build ./...` | U1, U2, U5 | Windows build succeeds |
| Lint | `staticcheck ./...` | All | No warnings |
| Phase 1 smoke | `scripts/phase1_verify.sh` | U10 | Runs 10s, stable render, no crashes |

**Quality gates for `/goal` / ce-work:** Each unit's test scenarios must pass. Phase 1 complete when integration verification shows live audio driving spectrum + metadata, no stutter, no line-printing.

## Definition of Done

### Global

- [ ] All Implementation Units U1–U10 complete per their Verification
- [ ] `go test ./... -race` passes
- [ ] `staticcheck ./...` clean
- [ ] Cross-compile to Windows succeeds
- [ ] Phase 1 integration verified: real Chrome/Tidal audio → visible spectrum + metadata, 60 FPS, no line-printing
- [ ] No abandoned experimental code in diff (cleanup criterion)

### Per-Unit

- **U1:** `go build` succeeds; `Makefile` targets work
- **U2:** Malgo backend captures loopback audio on Linux (CI) and Windows (manual); buffers at ~100 Hz
- **U3:** DSP pipeline produces correct log-binned spectrum for known inputs (sine, noise, silence)
- **U4:** Linux MPRIS extracts metadata from Spotify/Chrome; fallback works when no players
- **U5:** Windows build succeeds; metadata stub falls back gracefully (GSMTC cgo optional)
- **U6:** `docs/macos_setup.md` accurate and complete
- **U7:** `-race` clean; concurrent read/write safe
- **U8:** `View()` returns `tea.NewView` with `AltScreen=true`; zero `fmt.Print` in render path
- **U9:** All goroutines start/stop cleanly; flags parsed; quit on `q`
- **U10:** CI pipeline green; smoke script passes

## Appendix

### Platform Setup Matrix

| Platform | Loopback Capture | Metadata Hook | Setup Required |
|----------|------------------|---------------|----------------|
| Windows  | malgo WASAPI Loopback (native) | GSMTC (stub Phase 1, cgo Phase 2) | None (malgo handles WASAPI) |
| Linux    | malgo Pulse/PipeWire monitor | MPRIS (godbus/dbus) | Pulse/PipeWire running; user in `audio` group |
| macOS    | **Doc-only** (BlackHole + Multi-Output) | NowPlaying (stub) | Install BlackHole; create Multi-Output Device in Audio MIDI Setup |

### Key Constants (config/constants.go)

```go
const (
	SampleRate          = 48000           // Target; actual from device
	CapturePeriodMs     = 10              // → ~480 frames @ 48kHz
	CaptureBufferFrames = 1024            // Max; power of 2 for FFT
	FFTSize             = 1024            // Power of 2
	NumBins             = 60              // Visual columns
	MinFreqHz           = 30.0
	MaxFreqHz           = 16000.0
	TickRenderMs        = 16              // 60 FPS
	TickAnalyzeMs       = 33              // ~30 Hz FFT
	TickMetaMs          = 1500            // Metadata poll
	AGCTargetDbFS       = -6.0
	AGCAttackMs         = 100
	AGCReleaseMs        = 500
)
```

### CLI Flags (main.go)

```
-termvu
  -backend string     Audio backend: malgo (default)
  -device string      Capture device: auto (default), or device ID
  -bins int           Spectrum columns (default 60)
  -fps int            Render FPS (default 60)
  -help               Show help
```

### Risk Analysis

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| No Go GSMTC binding | High | Windows metadata unavailable Phase 1 | Stub + fallback label; cgo Phase 2 |
| malgo loopback unreliable on PipeWire | Medium | Linux capture fails | Fallback to explicit monitor device selection; test on PipeWire |
| DSP goroutine blocks render | Low | Stutter/frame drops | Decoupled tick cadences (KTD6); buffer channel size >2 |
| Terminal "prints lines" | High (if wrong) | Unusable UI | Hard constraint: `AltScreen=true`, `tea.NewView` only (KTD5) |
| Cross-platform build complexity | Medium | CI/maintenance burden | Build tags isolate CGO; CI tests Linux only |

### cliamp Pattern Mapping (for implementer)

| cliamp Path | termvu Adoption |
|-------------|-----------------|
| `ui/tick.go` constants | Copy → `ui/tick.go` |
| `ui/model/tick.go` `tickCmdAt` | Copy → `ui/tick.go` |
| `ui/model/view.go` `View() tea.View` + `AltScreen` | Copy → `ui/view.go` |
| `ui/fft.go` radix-2 + twiddles | Copy → `dsp/fft.go` |
| `ui/visualizer.go` spectrum rendering | Adapt → `ui/visualizer.go` |
| `mediactl/service_linux.go` DBus patterns | Inverse → `media/linux_mpris.go` (watch not export) |
| `internal/fileutil/atomic.go` | Adapt → `state/frame_buffer.go`, `metadata_store.go` |
| `player/tap.go` callback → channel | Adapt → `audio/malgo_backend.go` |

### Deferred to Follow-Up Work

- macOS loopback capture implementation (BlackHole runtime)
- Windows GSMTC cgo implementation
- Multiple visualizer modes (wave, scope, terrain, etc.)
- Custom animation physics (spring, decay, peak-hold gravity)
- Per-band EQ UI controls + config persistence
- Plugin/lua scripting (cliamp has this; explicitly out of scope)
- Daemon/IPC mode (cliamp has daemon; explicitly out of scope)