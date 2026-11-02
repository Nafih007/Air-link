//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

const webview2Download = "https://go.microsoft.com/fwlink/p/?LinkId=2124703"

func ensureWebView2() error {
	if webview2Present() {
		return nil
	}
	msg := "This app needs the Microsoft WebView2 Runtime (free).\n\nOpen the download page now?"
	if messageBoxYes(msg) {
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", webview2Download).Start()
	}
	return fmt.Errorf("WebView2 Runtime is not installed. Download: %s", webview2Download)
}

func webview2Present() bool {
	keys := []struct {
		root registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`},
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`},
		{registry.CURRENT_USER, `SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`},
	}
	for _, k := range keys {
		h, err := registry.OpenKey(k.root, k.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		pv, _, err := h.GetStringValue("pv")
		h.Close()
		if err == nil && pv != "" && pv != "0.0.0.0" {
			return true
		}
	}
	return false
}

func messageBoxYes(text string) bool {
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("MessageBoxW")
	title, _ := syscall.UTF16PtrFromString("Air-Link")
	body, _ := syscall.UTF16PtrFromString(text)
	r, _, _ := proc.Call(0, uintptr(unsafe.Pointer(body)), uintptr(unsafe.Pointer(title)), 0x04|0x20) // MB_YESNO | MB_ICONQUESTION
	return r == 6                                                                                     // IDYES
}
