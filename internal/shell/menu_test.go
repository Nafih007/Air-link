package shell

import "testing"

func TestVerbCommandAndIcon(t *testing.T) {
	exe := `C:\Program Files\Air-Link\airlink.exe`
	cmd := verbCommand(exe)
	if cmd != `"C:\Program Files\Air-Link\airlink.exe" "%1"` {
		t.Fatalf("cmd=%q", cmd)
	}
	icon := verbIcon(exe)
	if icon != `"C:\Program Files\Air-Link\airlink.exe",0` {
		t.Fatalf("icon=%q", icon)
	}
}

func TestParseExeFromCommand(t *testing.T) {
	exe := `C:\Program Files\Air-Link\airlink.exe`
	if got := parseExeFromCommand(`"` + exe + `" "%1"`); got != exe {
		t.Fatalf("got %q", got)
	}
	if got := parseExeFromCommand(`"` + exe + `" -- "%1"`); got != exe {
		t.Fatalf("legacy -- got %q", got)
	}
}

func TestSameExePath(t *testing.T) {
	a := `C:\Program Files\Air-Link\airlink.exe`
	b := `C:\PROGRAM FILES\Air-Link\airlink.exe`
	if !sameExePath(a, b) {
		t.Fatal("expected same")
	}
	if sameExePath(a, `C:\other\airlink.exe`) {
		t.Fatal("expected different")
	}
}

func TestShouldAutoRegister(t *testing.T) {
	if !ShouldAutoRegister(`C:\apps\airlink.exe`) {
		t.Fatal("portable exe")
	}
	if ShouldAutoRegister(`C:\Users\example\AppData\Local\go-build\b001\exe\airlink.exe`) {
		t.Fatal("go-build must be skipped")
	}
	if ShouldAutoRegister(`C:\apps\sendtophone.exe`) {
		t.Fatal("old exe name must not auto-register")
	}
	if ShouldAutoRegister(`C:\apps\other.exe`) {
		t.Fatal("wrong name")
	}
}

func TestMenuLabel(t *testing.T) {
	if menuLabel(MenuStatus{Installed: true}) != "Right-click menu: Installed ✓" {
		t.Fatal("installed")
	}
	if menuLabel(MenuStatus{NeedsRepair: true}) != "Right-click menu: Needs repair" {
		t.Fatal("repair")
	}
	if menuLabel(MenuStatus{}) != "Right-click menu: Not installed" {
		t.Fatal("missing")
	}
}
