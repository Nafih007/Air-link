package tray

import _ "embed"

//go:embed airlink.ico
var brandICO []byte

func appICO() []byte {
	if len(brandICO) > 0 {
		return brandICO
	}
	return blueDiscICO()
}

// blueDiscICO is a 16×16 32-bit ICO (no file on disk).
func blueDiscICO() []byte {
	const s = 16
	xor := make([]byte, s*s*4)
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			dx := float64(x) - 7.5
			dy := float64(y) - 7.5
			i := ((s-1-y)*s + x) * 4
			if dx*dx+dy*dy <= 49 {
				xor[i+0] = 0xfd
				xor[i+1] = 0x8b
				xor[i+2] = 0x3d
				xor[i+3] = 0xff
			}
		}
	}
	andRow := ((s + 31) / 32) * 4
	and := make([]byte, s*andRow)

	bihSize := 40
	xorOff := 6 + 16 + bihSize
	total := xorOff + len(xor) + len(and)
	b := make([]byte, total)
	// ICONDIR
	b[2] = 1
	b[4] = 1
	// ICONDIRENTRY
	b[6] = s
	b[7] = s
	b[10] = 1 // planes
	b[12] = 32
	putU32(b[14:], uint32(40+len(xor)+len(and)))
	putU32(b[18:], 22)
	// BITMAPINFOHEADER
	off := 22
	putU32(b[off:], 40)
	putU32(b[off+4:], uint32(s))
	putU32(b[off+8:], uint32(s*2))
	b[off+12] = 1
	b[off+14] = 32
	putU32(b[off+20:], uint32(len(xor)+len(and)))
	copy(b[xorOff:], xor)
	copy(b[xorOff+len(xor):], and)
	return b
}

func putU32(p []byte, v uint32) {
	p[0] = byte(v)
	p[1] = byte(v >> 8)
	p[2] = byte(v >> 16)
	p[3] = byte(v >> 24)
}
