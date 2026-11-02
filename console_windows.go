//go:build windows

package main

import (
	"os"
	"syscall"
)

func attachParentConsole() {
	mod := syscall.NewLazyDLL("kernel32.dll")
	r, _, _ := mod.NewProc("AttachConsole").Call(^uintptr(0)) // ATTACH_PARENT_PROCESS
	if r == 0 {
		mod.NewProc("AllocConsole").Call()
	}
	hout, err := syscall.Open("CONOUT$", syscall.O_RDWR, 0)
	if err == nil {
		os.Stdout = os.NewFile(uintptr(hout), "stdout")
		os.Stderr = os.NewFile(uintptr(hout), "stderr")
	}
	hin, err := syscall.Open("CONIN$", syscall.O_RDWR, 0)
	if err == nil {
		os.Stdin = os.NewFile(uintptr(hin), "stdin")
	}
}
