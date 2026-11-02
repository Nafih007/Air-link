//go:build windows

package shell

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const liveMutexName = `Local\AirLink.live`

var liveHandle windows.Handle

// ClaimLiveInstance returns true if this process owns the desktop instance.
func ClaimLiveInstance() (ok bool, err error) {
	name, err := windows.UTF16PtrFromString(liveMutexName)
	if err != nil {
		return false, err
	}
	h, err := windows.CreateMutex(nil, false, name)
	if err == windows.ERROR_ALREADY_EXISTS {
		if h != 0 {
			windows.CloseHandle(h)
		}
		return false, nil
	}
	if err != nil {
		return false, err
	}
	liveHandle = h
	return true, nil
}

// LiveInstanceExists reports whether a desktop instance already holds the mutex.
func LiveInstanceExists() bool {
	name, err := windows.UTF16PtrFromString(liveMutexName)
	if err != nil {
		return false
	}
	h, err := windows.OpenMutex(windows.SYNCHRONIZE, false, name)
	if err != nil {
		return false
	}
	windows.CloseHandle(h)
	return true
}

// ReleaseLiveInstance drops the mutex (process exit also releases it).
func ReleaseLiveInstance() {
	if liveHandle != 0 {
		windows.CloseHandle(liveHandle)
		liveHandle = 0
	}
}

// OfferPaths leaves selected paths for the live instance (Explorer multi-select / second launch).
func OfferPaths(paths []string) error {
	dir, err := coalesceDir()
	if err != nil {
		return err
	}
	return writePathDrop(dir, paths)
}

// PollOfferedPaths drains path drops from other processes.
func PollOfferedPaths() []string {
	dir, err := coalesceDir()
	if err != nil {
		return nil
	}
	extra, _ := readAndRemoveDrops(dir)
	return extra
}

// FocusMainWindow restores the Air-Link window if present.
func FocusMainWindow() {
	user32 := windows.NewLazySystemDLL("user32.dll")
	find := user32.NewProc("FindWindowW")
	title, _ := syscall.UTF16PtrFromString("Air-Link")
	hwnd, _, _ := find.Call(0, uintptr(unsafe.Pointer(title)))
	if hwnd == 0 {
		return
	}
	user32.NewProc("ShowWindow").Call(hwnd, 9) // SW_RESTORE
	user32.NewProc("SetForegroundWindow").Call(hwnd)
}
