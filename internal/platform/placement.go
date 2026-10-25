package platform

import "image"

// gap is the space between the anchor and the window, in physical pixels.
const gap = 6

// Place returns the top-left corner for a window of the given size next to
// anchor: just below it, or above it if it does not fit below. The result is
// clamped to work, so the window is fully visible whenever it fits.
// All values are physical screen pixels.
func Place(anchor image.Rectangle, size image.Point, work image.Rectangle) image.Point {
	x := anchor.Min.X
	y := anchor.Max.Y + gap
	if y+size.Y > work.Max.Y {
		y = anchor.Min.Y - gap - size.Y
	}
	return image.Pt(clamp(x, work.Min.X, work.Max.X-size.X), clamp(y, work.Min.Y, work.Max.Y-size.Y))
}

// clamp keeps v in [lo, hi]. If the range is empty (window larger than the
// work area), the top-left edge wins so the title bar stays reachable.
func clamp(v, lo, hi int) int {
	if v > hi {
		v = hi
	}
	if v < lo {
		v = lo
	}
	return v
}
