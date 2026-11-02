//go:build windows

package tray

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	wmHotkey = 0x0312
	wmQuit   = 0x0012
)

var (
	user32                 = windows.NewLazySystemDLL("user32.dll")
	procRegisterHotKey     = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey   = user32.NewProc("UnregisterHotKey")
	procGetMessageW        = user32.NewProc("GetMessageW")
	procPostThreadMessageW = user32.NewProc("PostThreadMessageW")
)

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
	Private uint32
}

func listenScreenshotHotkey(mods, vk uint, label string, onFire func(), onErr func(error)) (stop func()) {
	ready := make(chan uint32, 1)
	go func() {
		runtime.LockOSThread()
		tid := windows.GetCurrentThreadId()
		ready <- tid
		r, _, err := procRegisterHotKey.Call(0, 1, uintptr(mods), uintptr(vk))
		if r == 0 {
			onErr(fmt.Errorf("%s is already registered by another app: %v", label, err))
			return
		}
		defer procUnregisterHotKey.Call(0, 1)
		var m msg
		for {
			ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
			ir := int32(ret)
			if ir == 0 || ir == -1 {
				return
			}
			if m.Message == wmHotkey {
				onFire()
			}
		}
	}()
	tid := <-ready
	return func() {
		procPostThreadMessageW.Call(uintptr(tid), wmQuit, 0, 0)
	}
}

func warnHotkey(err error) {
	title, _ := syscall.UTF16PtrFromString("Air-Link")
	text, _ := syscall.UTF16PtrFromString(err.Error() + "\nScreenshot hotkey is off; tray Screenshot still works.")
	user32.NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), 0x30)
}
