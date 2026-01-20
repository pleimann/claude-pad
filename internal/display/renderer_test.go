package display

import (
	"testing"
)

func TestNewRenderer(t *testing.T) {
	r := NewRenderer(128, 64)

	if r.Width() != 128 {
		t.Errorf("Width() = %d, want 128", r.Width())
	}
	if r.Height() != 64 {
		t.Errorf("Height() = %d, want 64", r.Height())
	}
}

func TestRendererClear(t *testing.T) {
	r := NewRenderer(8, 8)

	// Set some pixels
	r.SetPixel(0, 0, true)
	r.SetPixel(7, 7, true)

	// Clear
	r.Clear()

	// Check all pixels are off
	data := r.GetFrameBuffer()
	for i, b := range data {
		if b != 0 {
			t.Errorf("byte %d = 0x%02X after Clear(), want 0x00", i, b)
		}
	}
}

func TestRendererSetPixel(t *testing.T) {
	r := NewRenderer(16, 8)

	// Set some pixels
	r.SetPixel(0, 0, true)
	r.SetPixel(7, 0, true)
	r.SetPixel(8, 0, true)
	r.SetPixel(15, 0, true)

	data := r.GetFrameBuffer()

	// First byte (pixels 0-7 of row 0): bits 7 and 0 should be set
	// MSB first: pixel 0 is bit 7, pixel 7 is bit 0
	if data[0] != 0x81 { // 10000001
		t.Errorf("byte 0 = 0x%02X, want 0x81", data[0])
	}

	// Second byte (pixels 8-15 of row 0): bits 7 and 0 should be set
	if data[1] != 0x81 { // 10000001
		t.Errorf("byte 1 = 0x%02X, want 0x81", data[1])
	}
}

func TestRendererSetPixelOff(t *testing.T) {
	r := NewRenderer(8, 8)

	// Set pixel on then off
	r.SetPixel(0, 0, true)
	r.SetPixel(0, 0, false)

	data := r.GetFrameBuffer()
	if data[0] != 0x00 {
		t.Errorf("byte 0 = 0x%02X after SetPixel(off), want 0x00", data[0])
	}
}

func TestRendererFillRect(t *testing.T) {
	r := NewRenderer(8, 4)

	// Fill a 4x2 rectangle at (2, 1)
	r.FillRect(2, 1, 4, 2)

	data := r.GetFrameBuffer()

	// Row 0: all zeros
	if data[0] != 0x00 {
		t.Errorf("row 0 = 0x%02X, want 0x00", data[0])
	}

	// Row 1: pixels 2-5 set (bits 5,4,3,2 in MSB-first)
	// 00111100 = 0x3C
	if data[1] != 0x3C {
		t.Errorf("row 1 = 0x%02X, want 0x3C", data[1])
	}

	// Row 2: same as row 1
	if data[2] != 0x3C {
		t.Errorf("row 2 = 0x%02X, want 0x3C", data[2])
	}

	// Row 3: all zeros
	if data[3] != 0x00 {
		t.Errorf("row 3 = 0x%02X, want 0x00", data[3])
	}
}

func TestRendererDrawRect(t *testing.T) {
	r := NewRenderer(8, 4)

	// Draw a 4x3 rectangle outline at (2, 0)
	r.DrawRect(2, 0, 4, 3)

	data := r.GetFrameBuffer()

	// Row 0: pixels 2-5 set (top edge)
	if data[0] != 0x3C {
		t.Errorf("row 0 = 0x%02X, want 0x3C", data[0])
	}

	// Row 1: pixels 2 and 5 set (left and right edges)
	// 00100100 = 0x24
	if data[1] != 0x24 {
		t.Errorf("row 1 = 0x%02X, want 0x24", data[1])
	}

	// Row 2: pixels 2-5 set (bottom edge)
	if data[2] != 0x3C {
		t.Errorf("row 2 = 0x%02X, want 0x3C", data[2])
	}
}

func TestRendererGetFrameBuffer(t *testing.T) {
	r := NewRenderer(16, 4)

	// 16 pixels wide = 2 bytes per row, 4 rows = 8 bytes total
	data := r.GetFrameBuffer()
	if len(data) != 8 {
		t.Errorf("len(data) = %d, want 8", len(data))
	}
}

func TestRendererGetRegion(t *testing.T) {
	r := NewRenderer(16, 8)

	// Set a pattern
	r.SetPixel(8, 2, true)  // This should be in region (8,2,8,4)
	r.SetPixel(15, 5, true) // This should also be in region

	// Get region starting at (8, 2), size 8x4
	region := r.GetRegion(8, 2, 8, 4)

	// 8 pixels wide = 1 byte per row, 4 rows = 4 bytes
	if len(region) != 4 {
		t.Fatalf("len(region) = %d, want 4", len(region))
	}

	// Row 0 of region (y=2 in full buffer): pixel 0 (x=8 in full) should be set
	// MSB first: pixel 0 is bit 7
	if region[0] != 0x80 {
		t.Errorf("region row 0 = 0x%02X, want 0x80", region[0])
	}

	// Row 3 of region (y=5 in full buffer): pixel 7 (x=15 in full) should be set
	// MSB first: pixel 7 is bit 0
	if region[3] != 0x01 {
		t.Errorf("region row 3 = 0x%02X, want 0x01", region[3])
	}
}

func TestRendererDrawText(t *testing.T) {
	r := NewRenderer(64, 16)

	// Just verify it doesn't panic
	r.DrawText(0, 13, "Hello")

	// Check that some pixels were set
	data := r.GetFrameBuffer()
	hasPixels := false
	for _, b := range data {
		if b != 0 {
			hasPixels = true
			break
		}
	}

	if !hasPixels {
		t.Error("DrawText() didn't set any pixels")
	}
}

func TestRendererDrawTextWrapped(t *testing.T) {
	r := NewRenderer(64, 32)

	// Draw wrapped text
	height := r.DrawTextWrapped(0, 13, 64, "Hello World Test")

	// Should return positive height
	if height <= 0 {
		t.Errorf("DrawTextWrapped() returned height %d, want > 0", height)
	}

	// Check that some pixels were set
	data := r.GetFrameBuffer()
	hasPixels := false
	for _, b := range data {
		if b != 0 {
			hasPixels = true
			break
		}
	}

	if !hasPixels {
		t.Error("DrawTextWrapped() didn't set any pixels")
	}
}

func TestSplitWords(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"hello world", []string{"hello", "world"}},
		{"  spaced  out  ", []string{"spaced", "out"}},
		{"single", []string{"single"}},
		{"", nil},
		{"a b c", []string{"a", "b", "c"}},
		{"tabs\tand\nnewlines", []string{"tabs", "and", "newlines"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitWords(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("splitWords(%q) = %v, want %v", tt.input, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitWords(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}
