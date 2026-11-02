//go:build windows

package tray

import (
	"sync"

	"github.com/getlantern/systray"
)

// Embedded is the in-process tray used when the Wails window hides.
type Embedded struct {
	OnShow       func()
	OnOpenFolder func()
	OnQuit       func()
}

var (
	embedMu   sync.Mutex
	embedOn   bool
	embedCb   Embedded
	embedQuit chan struct{}
)

// StartEmbedded shows a tray icon in this process (Windows). Safe to call once.
func StartEmbedded(cb Embedded) {
	embedMu.Lock()
	if embedOn {
		embedCb = cb
		embedMu.Unlock()
		return
	}
	embedOn = true
	embedCb = cb
	embedMu.Unlock()
	go systray.Run(onEmbedReady, func() {})
}

func onEmbedReady() {
	systray.SetIcon(appICO())
	systray.SetTitle("Air-Link")
	systray.SetTooltip("Air-Link")
	mShow := systray.AddMenuItem("Show window", "Bring the session window back")
	mFold := systray.AddMenuItem("Open received folder", "Downloads\\Air-Link")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Exit", "End session and quit")
	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				embedMu.Lock()
				fn := embedCb.OnShow
				embedMu.Unlock()
				if fn != nil {
					fn()
				}
			case <-mFold.ClickedCh:
				embedMu.Lock()
				fn := embedCb.OnOpenFolder
				embedMu.Unlock()
				if fn != nil {
					fn()
				}
			case <-mQuit.ClickedCh:
				embedMu.Lock()
				fn := embedCb.OnQuit
				embedMu.Unlock()
				if fn != nil {
					fn()
				}
				systray.Quit()
				return
			}
		}
	}()
}

// StopEmbedded removes the tray icon.
func StopEmbedded() {
	systray.Quit()
}
