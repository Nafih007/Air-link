package applog

import "testing"

func TestSanitizeTokenPath(t *testing.T) {
	in := "hit http://192.168.1.5:1234/t/0123456789abcdef0123456789abcdef/meta"
	out := Sanitize(in)
	if out == in {
		t.Fatal("token still present")
	}
	join := "scan http://192.168.1.5:9/j/0123456789abcdef0123456789abcdef/"
	if Sanitize(join) == join {
		t.Fatal("join code still present")
	}
	if Sanitize("plain") != "plain" {
		t.Fatal("plain")
	}
}
