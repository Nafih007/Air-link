package mdns

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

const mdnsAddr = "224.0.0.251:5353"

// Advertiser answers mDNS A queries for <hostname>.local on the LAN.
type Advertiser struct {
	mu   sync.Mutex
	conn *net.UDPConn
	name string // e.g. pc.local.
	ip   net.IP
	done chan struct{}
}

// Start joins 224.0.0.251:5353 on iface and answers A queries for hostname.local.
func Start(bindIP string) (*Advertiser, error) {
	ip := net.ParseIP(bindIP)
	if ip == nil || ip.To4() == nil {
		return nil, fmt.Errorf("mdns: need IPv4 bind address")
	}
	host, err := os.Hostname()
	if err != nil || host == "" {
		return nil, fmt.Errorf("mdns: hostname: %w", err)
	}
	if i := strings.IndexByte(host, '.'); i > 0 {
		host = host[:i]
	}
	host = strings.ToLower(host)

	iface := ifaceForIP(ip)
	gaddr, err := net.ResolveUDPAddr("udp4", mdnsAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenMulticastUDP("udp4", iface, gaddr)
	if err != nil {
		return nil, fmt.Errorf("mdns listen: %w", err)
	}
	_ = conn.SetReadBuffer(65535)

	a := &Advertiser{
		conn: conn,
		name: host + ".local.",
		ip:   ip.To4(),
		done: make(chan struct{}),
	}
	go a.loop()
	return a, nil
}

// HostnameLocal is the advertised name without a trailing dot (e.g. pc.local).
func (a *Advertiser) HostnameLocal() string {
	if a == nil {
		return ""
	}
	return strings.TrimSuffix(a.name, ".")
}

// Close leaves the multicast group.
func (a *Advertiser) Close() error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	select {
	case <-a.done:
	default:
		close(a.done)
	}
	if a.conn != nil {
		return a.conn.Close()
	}
	return nil
}

func (a *Advertiser) loop() {
	buf := make([]byte, 2048)
	for {
		n, src, err := a.conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-a.done:
				return
			default:
				return
			}
		}
		resp := answerA(buf[:n], a.name, a.ip)
		if resp == nil {
			continue
		}
		_, _ = a.conn.WriteToUDP(resp, src)
	}
}

func ifaceForIP(ip net.IP) *net.Interface {
	list, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, ifi := range list {
		addrs, _ := ifi.Addrs()
		for _, ad := range addrs {
			var cand net.IP
			switch v := ad.(type) {
			case *net.IPNet:
				cand = v.IP
			case *net.IPAddr:
				cand = v.IP
			}
			if cand != nil && cand.Equal(ip) {
				ifi := ifi
				return &ifi
			}
		}
	}
	return nil
}

func answerA(msg []byte, qname string, ip net.IP) []byte {
	if len(msg) < 12 {
		return nil
	}
	if msg[2]&0x80 != 0 {
		return nil // response, not a query
	}
	qd := binary.BigEndian.Uint16(msg[4:6])
	if qd == 0 {
		return nil
	}
	off := 12
	var name string
	name, off = readName(msg, off)
	if off < 0 || off+4 > len(msg) {
		return nil
	}
	qtype := binary.BigEndian.Uint16(msg[off : off+2])
	if qtype != 1 { // A
		return nil
	}
	want := strings.ToLower(qname)
	got := strings.ToLower(name)
	if !strings.HasSuffix(got, ".") {
		got += "."
	}
	if got != want {
		return nil
	}
	id := msg[0:2]
	out := make([]byte, 0, 64)
	out = append(out, id...)
	out = append(out, 0x84, 0x00) // response, authoritative
	out = append(out, 0, 0)       // QD
	out = append(out, 0, 1)       // AN
	out = append(out, 0, 0, 0, 0) // NS, AR
	out = appendQName(out, qname)
	rr := make([]byte, 10+4)
	binary.BigEndian.PutUint16(rr[0:2], 1)
	binary.BigEndian.PutUint16(rr[2:4], 1)
	binary.BigEndian.PutUint32(rr[4:8], 120)
	binary.BigEndian.PutUint16(rr[8:10], 4)
	copy(rr[10:], ip.To4())
	out = append(out, rr...)
	return out
}

func readName(msg []byte, off int) (string, int) {
	var b strings.Builder
	hops := 0
	end := off
	jumped := false
	for hops < 10 {
		if off >= len(msg) {
			return "", -1
		}
		l := int(msg[off])
		if l == 0 {
			off++
			if !jumped {
				end = off
			}
			break
		}
		if l&0xc0 == 0xc0 {
			if off+1 >= len(msg) {
				return "", -1
			}
			ptr := int(binary.BigEndian.Uint16(msg[off:off+2]) & 0x3fff)
			if !jumped {
				end = off + 2
			}
			off = ptr
			jumped = true
			hops++
			continue
		}
		off++
		if off+l > len(msg) {
			return "", -1
		}
		if b.Len() > 0 {
			b.WriteByte('.')
		}
		b.Write(msg[off : off+l])
		off += l
		if !jumped {
			end = off
		}
	}
	if b.Len() == 0 {
		return "", -1
	}
	b.WriteByte('.')
	return b.String(), end
}

func appendQName(dst []byte, name string) []byte {
	name = strings.TrimSuffix(name, ".")
	for _, lab := range strings.Split(name, ".") {
		dst = append(dst, byte(len(lab)))
		dst = append(dst, lab...)
	}
	return append(dst, 0)
}
