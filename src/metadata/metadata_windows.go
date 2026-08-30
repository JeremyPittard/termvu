//go:build windows

package metadata

import (
	"fmt"
	"time"

	"github.com/go-ole/go-ole"
	"github.com/saltosystems/winrt-go/windows/foundation"
	"github.com/saltosystems/winrt-go/windows/media/control"
)

const (
	// RO_INIT_MULTITHREADED
	roInitMultithreaded = 1
	// S_FALSE, returned by RoInitialize when the apartment is already initialized.
	sFalse = 0x1
)

const smtcSessionManagerClass = "Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager"

// GetMetadata queries the Windows System Media Transport Controls (SMTC) for
// the currently playing track. It returns title, artist, album, and playback
// state. When no media session is active it returns an error and the caller
// may treat it as "nothing playing".
func GetMetadata() (Metadata, error) {
	if err := ole.RoInitialize(roInitMultithreaded); err != nil {
		if oe, ok := err.(*ole.OleError); ok && oe.Code() == sFalse {
			// Already initialized on this thread; that is fine.
		} else if err != nil {
			return Metadata{}, fmt.Errorf("RoInitialize: %w", err)
		}
	}

	inspectable, err := ole.RoActivateInstance(smtcSessionManagerClass)
	if err != nil {
		return Metadata{}, fmt.Errorf("RoActivateInstance(%s): %w", smtcSessionManagerClass, err)
	}
	defer inspectable.Release()

	mgr := &control.GlobalSystemMediaTransportControlsSessionManager{
		IUnknown: inspectable.IUnknown,
	}

	session, err := mgr.GetCurrentSession()
	if err != nil {
		return Metadata{}, fmt.Errorf("GetCurrentSession: %w", err)
	}
	if session == nil {
		return Metadata{}, fmt.Errorf("no current media session")
	}
	defer session.Release()

	status, err := sessionPlaybackStatus(session)
	if err != nil {
		return Metadata{}, err
	}

	props, err := sessionMediaProperties(session)
	if err != nil {
		return Metadata{}, err
	}
	defer props.Release()

	title, err := props.GetTitle()
	if err != nil {
		title = ""
	}
	artist, err := props.GetArtist()
	if err != nil {
		artist = ""
	}
	album, err := props.GetAlbumTitle()
	if err != nil {
		album = ""
	}

	return Metadata{
		Title:   title,
		Artist:  artist,
		Album:   album,
		Playing: status == control.GlobalSystemMediaTransportControlsSessionPlaybackStatusPlaying,
	}, nil
}

func sessionPlaybackStatus(session *control.GlobalSystemMediaTransportControlsSession) (control.GlobalSystemMediaTransportControlsSessionPlaybackStatus, error) {
	info, err := session.GetPlaybackInfo()
	if err != nil {
		return 0, fmt.Errorf("GetPlaybackInfo: %w", err)
	}
	defer info.Release()
	return info.GetPlaybackStatus()
}

func sessionMediaProperties(session *control.GlobalSystemMediaTransportControlsSession) (*control.GlobalSystemMediaTransportControlsSessionMediaProperties, error) {
	op, err := session.TryGetMediaPropertiesAsync()
	if err != nil {
		return nil, fmt.Errorf("TryGetMediaPropertiesAsync: %w", err)
	}
	defer op.Release()

	// WinRT async operations for SMTC media properties complete quickly, but
	// we still wait for completion with a bounded poll.
	deadline := time.Now().Add(2 * time.Second)
	for {
		switch status, err := op.GetStatus(); {
		case err != nil:
			return nil, fmt.Errorf("async GetStatus: %w", err)
		case status == foundation.AsyncStatusCompleted:
			return mediaPropertiesFromResults(op)
		case status == foundation.AsyncStatusError || status == foundation.AsyncStatusCanceled:
			return nil, fmt.Errorf("async media properties ended with status %d", status)
		case time.Now().After(deadline):
			return nil, fmt.Errorf("timed out waiting for media properties")
		default:
			time.Sleep(2 * time.Millisecond)
		}
	}
}

func mediaPropertiesFromResults(op *foundation.IAsyncOperation) (*control.GlobalSystemMediaTransportControlsSessionMediaProperties, error) {
	result, err := op.GetResults()
	if err != nil {
		return nil, fmt.Errorf("async GetResults: %w", err)
	}
	inspectable := (*ole.IInspectable)(result)
	return &control.GlobalSystemMediaTransportControlsSessionMediaProperties{
		IUnknown: inspectable.IUnknown,
	}, nil
}