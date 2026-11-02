package wifi

import "testing"

func TestQRPayload(t *testing.T) {
	got := QRPayload(Credentials{SSID: "Cafe;WiFi", Password: `p\ass`, Auth: "WPA"})
	if got != `WIFI:T:WPA;S:Cafe\;WiFi;P:p\\ass;;` {
		t.Fatalf("%q", got)
	}
	open := QRPayload(Credentials{SSID: "Open", Password: ""})
	if open != "WIFI:T:nopass;S:Open;P:;;" {
		t.Fatalf("%q", open)
	}
}
