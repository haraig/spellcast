package platform

import (
	"image"
	"testing"
)

func TestPlace(t *testing.T) {
	work := image.Rect(0, 0, 1920, 1040)
	size := image.Pt(480, 320)
	tests := []struct {
		name   string
		anchor image.Rectangle
		work   image.Rectangle
		want   image.Point
	}{
		{"below the anchor", image.Rect(100, 100, 102, 120), work, image.Pt(100, 126)},
		{"above when no room below", image.Rect(100, 900, 102, 920), work, image.Pt(100, 900-gap-320)},
		{"clamped at the right edge", image.Rect(1800, 100, 1802, 120), work, image.Pt(1920-480, 126)},
		{"clamped at the left edge", image.Rect(-50, 100, -48, 120), work, image.Pt(0, 126)},
		{"mouse pointer 1x1 at the bottom right corner", image.Rect(1919, 1039, 1920, 1040), work, image.Pt(1440, 1039-gap-320)},
		{"secondary monitor left of the primary (negative coordinates)",
			image.Rect(-1500, 200, -1498, 220), image.Rect(-1920, 0, 0, 1040), image.Pt(-1500, 226)},
		{"secondary monitor, clamped at its right edge",
			image.Rect(-10, 200, -8, 220), image.Rect(-1920, 0, 0, 1040), image.Pt(-480, 226)},
		{"no room above or below: pinned to the top", image.Rect(100, 150, 102, 170), image.Rect(0, 0, 1920, 300), image.Pt(100, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Place(tt.anchor, size, tt.work)
			if got != tt.want {
				t.Errorf("Place = %v, want %v", got, tt.want)
			}
			r := image.Rectangle{Min: got, Max: got.Add(size)}
			if tt.work.Dx() >= size.X && tt.work.Dy() >= size.Y && !r.In(tt.work) {
				t.Errorf("window %v is not inside the work area %v", r, tt.work)
			}
		})
	}
}

func TestPlaceWindowLargerThanWorkArea(t *testing.T) {
	work := image.Rect(0, 0, 400, 300)
	got := Place(image.Rect(100, 100, 102, 120), image.Pt(480, 320), work)
	if got != image.Pt(0, 0) {
		t.Errorf("Place = %v, want top-left corner of the work area", got)
	}
}
