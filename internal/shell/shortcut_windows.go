//go:build windows

package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	ole32                = windows.NewLazySystemDLL("ole32.dll")
	procCoCreateInstance = ole32.NewProc("CoCreateInstance")

	clsidShellLink  = windows.GUID{Data1: 0x00021401, Data2: 0x0000, Data3: 0x0000, Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidIShellLinkW  = windows.GUID{Data1: 0x000214F9, Data2: 0x0000, Data3: 0x0000, Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidIPersistFile = windows.GUID{Data1: 0x0000010b, Data2: 0x0000, Data3: 0x0000, Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
)

const (
	clsctxInprocServer = 1
	rpcEChangedMode    = 0x80010106
)

type oleVtbl struct {
	queryInterface uintptr
	addRef         uintptr
	release        uintptr
}

type iShellLinkVtbl struct {
	oleVtbl
	getPath             uintptr
	getIDList           uintptr
	setIDList           uintptr
	getDescription      uintptr
	setDescription      uintptr
	getWorkingDirectory uintptr
	setWorkingDirectory uintptr
	getArguments        uintptr
	setArguments        uintptr
	getHotkey           uintptr
	setHotkey           uintptr
	getShowCmd          uintptr
	setShowCmd          uintptr
	getIconLocation     uintptr
	setIconLocation     uintptr
	setRelativePath     uintptr
	resolve             uintptr
	setPath             uintptr
}

type iPersistFileVtbl struct {
	oleVtbl
	getClassID    uintptr
	isDirty       uintptr
	load          uintptr
	save          uintptr
	saveCompleted uintptr
	getCurFile    uintptr
}

type iShellLink struct {
	vtbl *iShellLinkVtbl
}

type iPersistFile struct {
	vtbl *iPersistFileVtbl
}

func sendToShortcutPath() (string, error) {
	appdata, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appdata, "Microsoft", "Windows", "SendTo", sendToLnk), nil
}

func sendToShortcutOK(exe string) bool {
	p, err := sendToShortcutPath()
	if err != nil {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func installSendToShortcut(exe string) error {
	p, err := sendToShortcutPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	removeLegacySendToShortcuts()
	return createShortcut(p, exe)
}

func removeSendToShortcut() error {
	removeLegacySendToShortcuts()
	p, err := sendToShortcutPath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func removeLegacySendToShortcuts() {
	appdata, err := os.UserConfigDir()
	if err != nil {
		return
	}
	dir := filepath.Join(appdata, "Microsoft", "Windows", "SendTo")
	for _, name := range oldSendToNames {
		_ = os.Remove(filepath.Join(dir, name))
	}
}

func createShortcut(lnk, target string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	uninit := false
	if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED); err != nil {
		if !isHRESULT(err, 1) && !isHRESULT(err, rpcEChangedMode) {
			return fmt.Errorf("CoInitializeEx: %w", err)
		}
	} else {
		uninit = true
		defer windows.CoUninitialize()
	}
	_ = uninit

	var link *iShellLink
	hr, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidShellLink)),
		0,
		clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidIShellLinkW)),
		uintptr(unsafe.Pointer(&link)),
	)
	if hr != 0 {
		return fmt.Errorf("CoCreateInstance IShellLink: HRESULT 0x%X", hr)
	}
	defer oleRelease(unsafe.Pointer(link), link.vtbl.release)

	targetUTF16, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	if err := hresult(syscall.SyscallN(link.vtbl.setPath, uintptr(unsafe.Pointer(link)), uintptr(unsafe.Pointer(targetUTF16)))); err != nil {
		return fmt.Errorf("IShellLink.SetPath: %w", err)
	}
	wd, err := windows.UTF16PtrFromString(filepath.Dir(target))
	if err != nil {
		return err
	}
	if err := hresult(syscall.SyscallN(link.vtbl.setWorkingDirectory, uintptr(unsafe.Pointer(link)), uintptr(unsafe.Pointer(wd)))); err != nil {
		return fmt.Errorf("IShellLink.SetWorkingDirectory: %w", err)
	}
	if err := hresult(syscall.SyscallN(link.vtbl.setIconLocation, uintptr(unsafe.Pointer(link)), uintptr(unsafe.Pointer(targetUTF16)), 0)); err != nil {
		return fmt.Errorf("IShellLink.SetIconLocation: %w", err)
	}

	var persist *iPersistFile
	if err := hresult(syscall.SyscallN(link.vtbl.queryInterface, uintptr(unsafe.Pointer(link)), uintptr(unsafe.Pointer(&iidIPersistFile)), uintptr(unsafe.Pointer(&persist)))); err != nil {
		return fmt.Errorf("QueryInterface IPersistFile: %w", err)
	}
	defer oleRelease(unsafe.Pointer(persist), persist.vtbl.release)

	lnkUTF16, err := windows.UTF16PtrFromString(lnk)
	if err != nil {
		return err
	}
	if err := hresult(syscall.SyscallN(persist.vtbl.save, uintptr(unsafe.Pointer(persist)), uintptr(unsafe.Pointer(lnkUTF16)), 1)); err != nil {
		return fmt.Errorf("IPersistFile.Save: %w", err)
	}
	return nil
}

func oleRelease(obj unsafe.Pointer, release uintptr) {
	if obj == nil || release == 0 {
		return
	}
	_, _, _ = syscall.SyscallN(release, uintptr(obj))
}

func hresult(r0 uintptr, _ uintptr, _ error) error {
	if r0 == 0 {
		return nil
	}
	return fmt.Errorf("HRESULT 0x%X", r0)
}

func isHRESULT(err error, code uintptr) bool {
	errno, ok := err.(syscall.Errno)
	return ok && uintptr(errno) == code
}
