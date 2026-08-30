//go:build windows

package audio

import (
	"context"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

// AudioCapture captures the system audio output using WASAPI loopback and
// computes the spectrum.
type AudioCapture struct {
	mu         sync.Mutex
	spectrum   Spectrum
	an         *analyzer
	sampleRate float64
	fftSize    int

	ctx    context.Context
	cancel context.CancelFunc

	audioClient   *wca.IAudioClient
	captureClient *wca.IAudioCaptureClient
	channels      int
}

const (
	// ERole eConsole is 0; EDataFlow eRender is wca.ERender (0).
	eConsoleRole = 0
)

// NewAudioCapture creates a WASAPI loopback capture for the default render
// endpoint. The requested sample rate is ignored on Windows because loopback
// must run at the endpoint's mix-format rate.
func NewAudioCapture(sampleRate float64, fftSize int) (*AudioCapture, error) {
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return nil, fmt.Errorf("CoInitializeEx: %w", err)
	}

	var enumerator *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(
		wca.CLSID_MMDeviceEnumerator,
		0,
		wca.CLSCTX_ALL,
		wca.IID_IMMDeviceEnumerator,
		&enumerator,
	); err != nil {
		return nil, fmt.Errorf("create MMDeviceEnumerator: %w", err)
	}
	defer enumerator.Release()

	var device *wca.IMMDevice
	if err := enumerator.GetDefaultAudioEndpoint(wca.ERender, eConsoleRole, &device); err != nil {
		return nil, fmt.Errorf("get default render endpoint: %w", err)
	}
	defer device.Release()

	var audioClient *wca.IAudioClient
	if err := device.Activate(wca.IID_IAudioClient, wca.CLSCTX_ALL, nil, &audioClient); err != nil {
		return nil, fmt.Errorf("activate IAudioClient: %w", err)
	}

	var wfx *wca.WAVEFORMATEX
	if err := audioClient.GetMixFormat(&wfx); err != nil {
		audioClient.Release()
		return nil, fmt.Errorf("GetMixFormat: %w", err)
	}
	defer ole.CoTaskMemFree(uintptr(unsafe.Pointer(wfx)))

	// Use a buffer duration matching the endpoint's default period.
	var defaultPeriod, minPeriod wca.REFERENCE_TIME
	if err := audioClient.GetDevicePeriod(&defaultPeriod, &minPeriod); err != nil {
		defaultPeriod = 10_000_000 // 1 second fallback (100ns units)
	}

	if err := audioClient.Initialize(
		wca.AUDCLNT_SHAREMODE_SHARED,
		wca.AUDCLNT_STREAMFLAGS_LOOPBACK,
		defaultPeriod,
		0,
		wfx,
		nil,
	); err != nil {
		audioClient.Release()
		return nil, fmt.Errorf("audio client Initialize (loopback): %w", err)
	}

	var captureClient *wca.IAudioCaptureClient
	if err := audioClient.GetService(wca.IID_IAudioCaptureClient, &captureClient); err != nil {
		audioClient.Release()
		return nil, fmt.Errorf("GetService IAudioCaptureClient: %w", err)
	}

	mixRate := float64(wfx.NSamplesPerSec)
	if mixRate == 0 {
		mixRate = 48000
	}

	return &AudioCapture{
		sampleRate:    mixRate,
		fftSize:       fftSize,
		audioClient:   audioClient,
		captureClient: captureClient,
		channels:      int(wfx.NChannels),
		an:            newAnalyzer(mixRate, fftSize),
	}, nil
}

// Start begins the loopback read loop and sends spectrum updates on the
// returned channel. It stops when the context is canceled.
func (ac *AudioCapture) Start(parentCtx context.Context) <-chan Spectrum {
	ac.ctx, ac.cancel = context.WithCancel(parentCtx)

	ch := make(chan Spectrum)
	go func() {
		defer close(ch)
		defer ac.captureClient.Release()
		defer ac.audioClient.Release()

		if err := ac.audioClient.Start(); err != nil {
			return
		}
		defer ac.audioClient.Stop()

		ac.readLoop(ch)
	}()
	return ch
}

// readLoop polls the capture client for packets, downmixes to mono, and feeds
// the spectrum analyzer.
func (ac *AudioCapture) readLoop(ch chan<- Spectrum) {
	var data *byte
	var numFrames, flags uint32
	channels := ac.channels

	for {
		select {
		case <-ac.ctx.Done():
			return
		default:
		}

		var packetSize uint32
		if err := ac.captureClient.GetNextPacketSize(&packetSize); err != nil {
			time.Sleep(time.Millisecond)
			continue
		}
		if packetSize == 0 {
			time.Sleep(time.Millisecond)
			continue
		}

		if err := ac.captureClient.GetBuffer(&data, &numFrames, &flags, nil, nil); err != nil {
			time.Sleep(time.Millisecond)
			continue
		}

		if numFrames > 0 {
			samples := float32ToMono(unsafe.Pointer(data), int(numFrames), channels)
			ac.mu.Lock()
			if s := ac.an.push(samples); s != nil {
				ac.spectrum = *s
			}
			ac.mu.Unlock()
		}

		_ = ac.captureClient.ReleaseBuffer(numFrames)

		select {
		case <-ac.ctx.Done():
			return
		case ch <- ac.readSpectrum():
		}
	}
}

func (ac *AudioCapture) readSpectrum() Spectrum {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	return ac.spectrum
}

// float32ToMono converts an interleaved float32 buffer to mono float64 by
// averaging the channels of each frame.
func float32ToMono(ptr unsafe.Pointer, frames, channels int) []float64 {
	if channels <= 0 {
		channels = 1
	}
	total := frames * channels
	buf := unsafe.Slice((*float32)(ptr), total)
	out := make([]float64, frames)
	for f := 0; f < frames; f++ {
		var sum float64
		for c := 0; c < channels; c++ {
			sum += float64(buf[f*channels+c])
		}
		out[f] = sum / float64(channels)
	}
	return out
}

// Stop stops the audio capture.
func (ac *AudioCapture) Stop() {
	if ac.cancel != nil {
		ac.cancel()
	}
}