//go:build ignore

// gen-icon renders the Tallywell application icon as a 1024×1024 PNG at
// dist/tallywell-icon-1024.png. It is called by scripts/build-mac-app.sh.
// Run standalone: go run scripts/gen-icon.go
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
)

func main() {
	const size = 1024
	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	// App accent colour — calm forest green, matches the web UI.
	bg := color.NRGBA{R: 0x3b, G: 0x6d, B: 0x57, A: 0xff}
	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}

	// macOS app icon corner radius: Apple's HIG recommends ~22.37% of size.
	radius := float64(size) * 0.2237

	// Rounded rectangle background with 1-pixel anti-aliased edge.
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			alpha := roundedRectAlpha(float64(x)+0.5, float64(y)+0.5, float64(size), radius)
			if alpha > 0 {
				img.SetNRGBA(x, y, color.NRGBA{R: bg.R, G: bg.G, B: bg.B, A: uint8(alpha)})
			}
		}
	}

	// Tally mark proportions (at 1024 × 1024).
	const (
		markW = 42  // width of each vertical bar
		markH = 360 // height of each vertical bar
		gap   = 84  // centre-to-centre spacing
	)
	groupW := 3*gap + markW
	startX := (size-groupW)/2
	startY := (size-markH)/2

	// Four vertical tally bars.
	for i := 0; i < 4; i++ {
		cx := startX + i*gap + markW/2
		fillRect(img, cx-markW/2, startY, markW, markH, white)
	}

	// Diagonal strike-through: ~44° angle across all four bars.
	drawThickLine(img,
		float64(startX-82), float64(startY+markH+42), // bottom-left
		float64(startX+groupW+82), float64(startY-42), // top-right
		42, white)

	os.MkdirAll("dist", 0o755)
	f, err := os.Create("dist/tallywell-icon-1024.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
	log.Println("wrote dist/tallywell-icon-1024.png")
}

// roundedRectAlpha returns the coverage (0–255) of pixel centre (px,py) inside
// a rounded square of the given size and corner radius. The 1-pixel feather
// at the edge gives smooth anti-aliased corners.
func roundedRectAlpha(px, py, size, r float64) float64 {
	// Corner arc centres.
	cx1, cy1 := r, r
	cx2, cy2 := size-r, r
	cx3, cy3 := r, size-r
	cx4, cy4 := size-r, size-r

	var d float64 // signed distance: negative = inside, positive = outside
	switch {
	case px < cx1 && py < cy1:
		d = hypot(px-cx1, py-cy1) - r
	case px > cx2 && py < cy2:
		d = hypot(px-cx2, py-cy2) - r
	case px < cx3 && py > cy3:
		d = hypot(px-cx3, py-cy3) - r
	case px > cx4 && py > cy4:
		d = hypot(px-cx4, py-cy4) - r
	default:
		// Inside the straight edges — fully covered.
		return 255
	}
	if d <= 0 {
		return 255
	}
	if d < 1 {
		return (1 - d) * 255
	}
	return 0
}

func hypot(dx, dy float64) float64 { return math.Sqrt(dx*dx + dy*dy) }

func fillRect(img *image.NRGBA, x, y, w, h int, c color.NRGBA) {
	b := img.Bounds()
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			if px >= b.Min.X && px < b.Max.X && py >= b.Min.Y && py < b.Max.Y {
				img.SetNRGBA(px, py, c)
			}
		}
	}
}

// drawThickLine draws a filled capsule-capped line from (x1,y1) to (x2,y2)
// with the given diameter, anti-aliased at the edges.
func drawThickLine(img *image.NRGBA, x1, y1, x2, y2, thick float64, c color.NRGBA) {
	dx, dy := x2-x1, y2-y1
	length := hypot(dx, dy)
	if length == 0 {
		return
	}
	half := thick / 2
	minX := int(math.Min(x1, x2) - half - 2)
	maxX := int(math.Max(x1, x2) + half + 2)
	minY := int(math.Min(y1, y2) - half - 2)
	maxY := int(math.Max(y1, y2) + half + 2)
	b := img.Bounds()

	for py := minY; py <= maxY; py++ {
		for px := minX; px <= maxX; px++ {
			if px < b.Min.X || px >= b.Max.X || py < b.Min.Y || py >= b.Max.Y {
				continue
			}
			fx, fy := float64(px)+0.5, float64(py)+0.5
			// Project onto segment, clamped (capsule ends).
			t := ((fx-x1)*dx + (fy-y1)*dy) / (length * length)
			if t < 0 {
				t = 0
			} else if t > 1 {
				t = 1
			}
			d := hypot(fx-(x1+t*dx), fy-(y1+t*dy)) - half
			var alpha float64
			if d <= 0 {
				alpha = 255
			} else if d < 1 {
				alpha = (1 - d) * 255
			} else {
				continue
			}
			// Blend onto existing pixel (background stays underneath).
			existing := img.NRGBAAt(px, py)
			a := uint8(alpha)
			img.SetNRGBA(px, py, color.NRGBA{
				R: blend(existing.R, c.R, a),
				G: blend(existing.G, c.G, a),
				B: blend(existing.B, c.B, a),
				A: maxU8(existing.A, a),
			})
		}
	}
}

func blend(bg, fg, alpha uint8) uint8 {
	a := float64(alpha) / 255
	return uint8(float64(fg)*a + float64(bg)*(1-a))
}

func maxU8(a, b uint8) uint8 {
	if a > b {
		return a
	}
	return b
}
