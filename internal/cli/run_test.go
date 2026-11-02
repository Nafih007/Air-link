package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestWantsCLI(t *testing.T) {
	if WantsCLI(nil) || WantsCLI([]string{`C:\a.txt`}) {
		t.Fatal("files only should be GUI")
	}
	if !WantsCLI([]string{"-receive"}) || !WantsCLI([]string{"--help"}) {
		t.Fatal("flags should be CLI")
	}
}

func TestVersionFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"--version"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "Air-Link v") {
		t.Fatalf("stdout=%q", out.String())
	}
}

func TestHelpFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"--help"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%q", code, errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "Usage:") {
		t.Fatalf("help=%q", errBuf.String())
	}
}

func TestHelpMentionsRoom(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"--help"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(errBuf.String(), "No args") {
		t.Fatalf("help=%q", errBuf.String())
	}
}

func TestListIfaces(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"-list-ifaces"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "IPv4") && !strings.Contains(out.String(), "No IPv4") {
		t.Fatalf("stdout=%q", out.String())
	}
}
