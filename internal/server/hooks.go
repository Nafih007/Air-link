package server

import (
	"sync"
	"sync/atomic"
	"time"

	"sendtophone/internal/pair"
)

// Hooks let the UI/CLI react to session activity without seeing the token.
type Hooks struct {
	// OnPhoneHit fires on any authenticated page/download request (once or more).
	OnPhoneHit func()
	// OnTransferComplete fires after a download handler finishes.
	OnTransferComplete func()
	// OnProgress fires during a download with bytes written (total may be -1 if unknown).
	OnProgress func(written, total int64)
	// OnUploadPending fires when the phone sent files and PC approval is required.
	OnUploadPending func(id string, fileCount int, totalSize int64, hasText bool)
	// OnUploadSaved fires after files were written to the upload folder.
	OnUploadSaved func(fileCount int, totalSize int64)
	// OnUploadProgress fires while chunked uploads arrive (bytes received / total).
	OnUploadProgress func(got, total int64)
	// OnClipboardText fires when the phone sent text that should go on the PC clipboard.
	OnClipboardText func(text string)
	// OnDevicesChanged fires when a phone is seen, allowed, denied, or kicked.
	OnDevicesChanged func(devices []Device)
	// OnJoinConsumed fires after a join code is used or rotated. No secrets.
	OnJoinConsumed func()
	// OnPairsChanged fires after a phone is paired, renamed, or removed.
	OnPairsChanged func(devices []pair.Record)
}

type activity struct {
	phoneHit  atomic.Bool
	transfers atomic.Int64
	mu        sync.Mutex
	last      time.Time
}

func (s *Server) noteHit() {
	s.act.phoneHit.Store(true)
	s.act.mu.Lock()
	s.act.last = time.Now()
	s.act.mu.Unlock()
	if s.cfg.Hooks.OnPhoneHit != nil {
		s.cfg.Hooks.OnPhoneHit()
	}
}

func (s *Server) noteTransferComplete() {
	s.act.transfers.Add(1)
	s.act.mu.Lock()
	s.act.last = time.Now()
	s.act.mu.Unlock()
	if s.cfg.Hooks.OnTransferComplete != nil {
		s.cfg.Hooks.OnTransferComplete()
	}
}

// PhoneHit reports whether any authenticated request arrived.
func (s *Server) PhoneHit() bool {
	return s.act.phoneHit.Load()
}

// LastActivity is the last authenticated request time (zero if none).
func (s *Server) LastActivity() time.Time {
	s.act.mu.Lock()
	defer s.act.mu.Unlock()
	return s.act.last
}

// TransferCount is how many download handlers completed.
func (s *Server) TransferCount() int64 {
	return s.act.transfers.Load()
}
