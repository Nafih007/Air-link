package shell

import (
	"path/filepath"
	"strings"
	"time"
)

const (
	verbName    = "AirLink"
	verbLabel   = "Send with Air-Link"
	oldVerbName = "SendToPhoneQR"
	legacyVerb  = "SendToPhone"
	sendToLnk   = "Air-Link.lnk"

	// MultiLaunchWindow is how long Explorer's per-file launches are batched.
	MultiLaunchWindow = 500 * time.Millisecond
)

var oldSendToNames = []string{
	"Send to phone.lnk",
	"Phone (QR).lnk",
}

var hidingValueNames = []string{
	"Extended",
	"LegacyDisable",
	"ProgrammaticAccessOnly",
	"ExplorerCommandHandler",
}

// MenuStatus is shown in the desktop Settings panel.
type MenuStatus struct {
	Installed     bool   `json:"installed"`
	NeedsRepair   bool   `json:"needsRepair"`
	CurrentExe    string `json:"currentExe"`
	RegisteredExe string `json:"registeredExe"`
	SendToOK      bool   `json:"sendToOK"`
	Label         string `json:"label"`
	Win11Note     string `json:"win11Note"`
	Detail        string `json:"detail"`
}

func win11SettingsNote() string {
	return "On Windows 11 this entry appears under “Show more options” (or hold Shift and right-click). The short Windows 11 menu needs a signed package; that is a later task."
}

func verbCommand(exe string) string {
	return `"` + exe + `" "%1"`
}

func verbIcon(exe string) string {
	return `"` + exe + `",0`
}

func parseExeFromCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ""
	}
	if strings.HasPrefix(cmd, `"`) {
		end := strings.Index(cmd[1:], `"`)
		if end >= 0 {
			return cmd[1 : 1+end]
		}
	}
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func sameExePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
	}
	return strings.EqualFold(filepath.Clean(absA), filepath.Clean(absB))
}

// ShouldAutoRegister is false for `go run` temp binaries so we do not
// point Explorer at a path that disappears.
func ShouldAutoRegister(exe string) bool {
	if exe == "" {
		return false
	}
	base := filepath.Base(exe)
	if !strings.EqualFold(base, "airlink.exe") {
		return false
	}
	lower := strings.ToLower(filepath.ToSlash(exe))
	if strings.Contains(lower, "/go-build") || strings.Contains(lower, "/go-tmp") {
		return false
	}
	return true
}

func menuLabel(st MenuStatus) string {
	if st.Installed && !st.NeedsRepair {
		return "Right-click menu: Installed ✓"
	}
	if st.NeedsRepair {
		return "Right-click menu: Needs repair"
	}
	return "Right-click menu: Not installed"
}
