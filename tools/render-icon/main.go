// Command render-icon renders an SVG to a PNG with a transparent background.
//
// Usage: go run . <in.svg> <out.png> [size]
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: render-icon <in.svg> <out.png> [size]")
		os.Exit(2)
	}
	size := 1024
	if len(os.Args) > 3 {
		n, err := strconv.Atoi(os.Args[3])
		if err != nil || n <= 0 {
			fmt.Fprintln(os.Stderr, "invalid size")
			os.Exit(2)
		}
		size = n
	}
	if err := render(os.Args[1], os.Args[2], size); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func render(in, out string, size int) error {
	icon, err := oksvg.ReadIcon(in, oksvg.WarnErrorMode)
	if err != nil {
		return err
	}
	icon.SetTarget(0, 0, float64(size), float64(size))
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	icon.Draw(rasterx.NewDasher(size, size, rasterx.NewScannerGV(size, size, img, img.Bounds())), 1)

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
