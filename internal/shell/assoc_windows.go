//go:build windows

package shell

import "syscall"

const (
	shcneAssocChanged = 0x08000000
	shcnfIDList       = 0x0000
)

func notifyAssocChanged() {
	mod := syscall.NewLazyDLL("shell32.dll")
	proc := mod.NewProc("SHChangeNotify")
	_, _, _ = proc.Call(uintptr(shcneAssocChanged), uintptr(shcnfIDList), 0, 0)
}
