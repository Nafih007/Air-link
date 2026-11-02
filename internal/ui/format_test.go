package ui

import "testing"

func TestFormatSize(t *testing.T) {
	cases := map[int64]string{
		0:               "0 bytes",
		512:             "512 bytes",
		2048:            "2.0 KB",
		3 * 1024 * 1024: "3.0 MB",
	}
	for n, want := range cases {
		if got := FormatSize(n); got != want {
			t.Errorf("FormatSize(%d)=%q want %q", n, got, want)
		}
	}
}
