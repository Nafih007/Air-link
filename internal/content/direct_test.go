package content

import "testing"

func TestFitsDirectQR(t *testing.T) {
	if !FitsDirectQR("https://example.com") {
		t.Fatal("short url")
	}
	long := make([]rune, MaxDirectQRChars+1)
	for i := range long {
		long[i] = 'a'
	}
	if FitsDirectQR(string(long)) {
		t.Fatal("too long")
	}
	if FitsDirectQR("   ") {
		t.Fatal("blank")
	}
}

func TestLooksLikeURL(t *testing.T) {
	if !LooksLikeURL("https://example.com/x") {
		t.Fatal("https")
	}
	if LooksLikeURL("example.com") || LooksLikeURL("ftp://x") {
		t.Fatal("not http url")
	}
}
