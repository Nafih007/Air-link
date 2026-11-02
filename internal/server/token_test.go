package server

import "testing"

func TestNewTokenLengthAndHex(t *testing.T) {
	tok, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(tok) != 32 {
		t.Fatalf("len=%d want 32", len(tok))
	}
	for _, c := range tok {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Fatalf("non-hex %q in %q", c, tok)
		}
	}
}

func TestTokenValid(t *testing.T) {
	tok, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if !TokenValid(tok, tok) {
		t.Fatal("same token should match")
	}
	if TokenValid(tok, tok[:len(tok)-1]+"0") && tok[len(tok)-1] == '0' {
		// flip last nibble carefully
	}
	other, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if TokenValid(tok, other) && tok != other {
		t.Fatal("different tokens must not match")
	}
	if TokenValid("", "") || TokenValid(tok, "") || TokenValid("", tok) {
		t.Fatal("empty must not match")
	}
	if TokenValid(tok, tok+"a") {
		t.Fatal("length mismatch must not match")
	}
}

func TestExtractToken(t *testing.T) {
	tok := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	got, ok := extractToken("/t/" + tok + "/")
	if !ok || got != tok {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	got, ok = extractToken("/t/" + tok + "/download")
	if !ok || got != tok {
		t.Fatalf("subpath got %q ok=%v", got, ok)
	}
	if _, ok := extractToken("/"); ok {
		t.Fatal("root should fail")
	}
	if _, ok := extractToken("/t/"); ok {
		t.Fatal("empty token should fail")
	}
	if _, ok := extractToken("/t/short/"); ok {
		t.Fatal("short token should fail")
	}
	short := "0123456789abcdef0123456789abcdef"
	if _, ok := extractToken("/t/" + short + "/"); ok {
		t.Fatal("32-char session tokens must not match device tokens")
	}
	if _, ok := extractToken("/x/" + tok + "/"); ok {
		t.Fatal("wrong prefix should fail")
	}
}
