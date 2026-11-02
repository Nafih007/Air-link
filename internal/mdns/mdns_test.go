package mdns

import (
	"encoding/binary"
	"net"
	"testing"
)

func TestAnswerAMatch(t *testing.T) {
	q := queryA("office.local.")
	ip := net.ParseIP("192.168.1.9").To4()
	resp := answerA(q, "office.local.", ip)
	if resp == nil {
		t.Fatal("expected answer")
	}
	if resp[2]&0x80 == 0 {
		t.Fatal("not a response")
	}
	if binary.BigEndian.Uint16(resp[6:8]) != 1 {
		t.Fatalf("ancount %d", binary.BigEndian.Uint16(resp[6:8]))
	}
	if string(resp[len(resp)-4:]) != string(ip) {
		t.Fatalf("rdata %v", resp[len(resp)-4:])
	}
}

func TestAnswerAOtherName(t *testing.T) {
	q := queryA("other.local.")
	if answerA(q, "office.local.", net.IPv4(192, 168, 1, 9)) != nil {
		t.Fatal("should ignore other names")
	}
}

func queryA(name string) []byte {
	b := []byte{0x12, 0x34, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	b = appendQName(b, name)
	typclass := make([]byte, 4)
	binary.BigEndian.PutUint16(typclass[0:2], 1)
	binary.BigEndian.PutUint16(typclass[2:4], 1)
	return append(b, typclass...)
}
