//go:build windows

package ui

import (
	"fmt"

	"github.com/jchv/go-webview2"
)

// RunSessionWindow opens a WebView2 window with the QR and session info.
// It blocks until the window is closed. Closing the window returns nil.
func RunSessionWindow(s Session, ctrl *Control) error {
	html, err := buildSessionHTML(s)
	if err != nil {
		return err
	}
	if ctrl == nil {
		ctrl = NewControl()
	}

	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  "Air-Link",
			Width:  380,
			Height: 700,
			Center: true,
		},
	})
	if w == nil {
		return fmt.Errorf("failed to create WebView2 window (is the WebView2 Runtime installed?)")
	}
	defer w.Destroy()

	w.SetSize(380, 700, webview2.HintFixed)
	ctrl.Attach(
		func(js string) { w.Dispatch(func() { w.Eval(js) }) },
		func() { w.Terminate() },
	)
	if s.OnExpired != nil {
		_ = w.Bind("sessionExpired", func() {
			s.OnExpired()
			w.Terminate()
		})
	}
	_ = w.Bind("uploadDecide", func(id string, accept bool) {
		ctrl.UploadDecide(id, accept)
	})
	_ = w.Bind("openUploadFolder", func() {
		ctrl.OpenFolderClicked()
	})
	_ = w.Bind("kickDevice", func(id string) {
		ctrl.KickClicked(id)
	})
	_ = w.Bind("renamePair", func(id, name string) {
		ctrl.PairRenameClicked(id, name)
	})
	_ = w.Bind("removePair", func(id string) {
		ctrl.PairRemoveClicked(id)
	})
	w.SetHtml(html)
	w.Run()
	return nil
}
