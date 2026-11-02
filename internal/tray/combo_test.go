package tray

import "testing"

func TestParseComboDefault(t *testing.T) {
	mods, vk, label, err := ParseCombo("Ctrl+Shift+P")
	if err != nil {
		t.Fatal(err)
	}
	if vk != 0x50 {
		t.Fatalf("vk %x", vk)
	}
	if mods&modControl == 0 || mods&modShift == 0 {
		t.Fatalf("mods %x", mods)
	}
	if label != "Ctrl+Shift+P" {
		t.Fatalf("label %q", label)
	}
}

func TestParseComboBad(t *testing.T) {
	if _, _, _, err := ParseCombo("Ctrl"); err == nil {
		t.Fatal("expected error")
	}
	if _, _, _, err := ParseCombo("Ctrl+Shift+P+Q"); err == nil {
		t.Fatal("expected error")
	}
}
