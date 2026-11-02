package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

func main() {
	const s = 256
	img := image.NewRGBA(image.Rect(0, 0, s, s))
	bg := color.RGBA{15, 20, 25, 255}
	phone := color.RGBA{61, 139, 253, 255}
	screen := color.RGBA{15, 20, 25, 255}
	arrow := color.RGBA{231, 236, 243, 255}
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			img.Set(x, y, bg)
		}
	}
	fillRound(img, 78, 28, 178, 228, 22, phone)
	fillRound(img, 92, 48, 164, 188, 10, screen)
	fillRound(img, 118, 198, 138, 210, 6, screen)
	// arrow shaft
	for y := 118; y < 138; y++ {
		for x := 36; x < 118; x++ {
			img.Set(x, y, arrow)
		}
	}
	// arrow head
	for i := 0; i < 28; i++ {
		for y := 108 + i/2; y < 148-i/2; y++ {
			img.Set(118+i, y, arrow)
		}
	}
	out := filepath.Join("build", "appicon.png")
	_ = os.MkdirAll("build", 0o755)
	f, err := os.Create(out)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func fillRound(img *image.RGBA, x0, y0, x1, y1, r int, c color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if insideRound(x, y, x0, y0, x1, y1, r) {
				img.Set(x, y, c)
			}
		}
	}
}

func insideRound(x, y, x0, y0, x1, y1, r int) bool {
	cx, cy := x, y
	if x < x0+r && y < y0+r {
		dx, dy := x0+r-x, y0+r-y
		return dx*dx+dy*dy <= r*r
	}
	if x >= x1-r && y < y0+r {
		dx, dy := x-(x1-r-1), y0+r-y
		return dx*dx+dy*dy <= r*r
	}
	if x < x0+r && y >= y1-r {
		dx, dy := x0+r-x, y-(y1-r-1)
		return dx*dx+dy*dy <= r*r
	}
	if x >= x1-r && y >= y1-r {
		dx, dy := x-(x1-r-1), y-(y1-r-1)
		return dx*dx+dy*dy <= r*r
	}
	_ = cx
	_ = cy
	return x >= x0 && x < x1 && y >= y0 && y < y1
}
