package main

import (
	"bytes"
	"strings"
	"testing"

	"sendtophone/internal/cli"
)

func TestVersionFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := cli.Run([]string{"--version"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "Air-Link v") {
		t.Fatalf("stdout=%q", out.String())
	}
}

func TestHelpFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := cli.Run([]string{"--help"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%q", code, errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "Usage:") {
		t.Fatalf("help=%q", errBuf.String())
	}
}
