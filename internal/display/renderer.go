package display

import (
	"image"
	"image/color"
	"image/draw"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// Renderer renders text and graphics to a 1-bit frame buffer
type Renderer struct {
	width  int
	height int
	img    *image.Gray
	face   font.Face
}

// NewRenderer creates a new display renderer
func NewRenderer(width, height int) *Renderer {
	return &Renderer{
		width:  width,
		height: height,
		img:    image.NewGray(image.Rect(0, 0, width, height)),
		face:   basicfont.Face7x13,
	}
}

// Clear clears the frame buffer
func (r *Renderer) Clear() {
	draw.Draw(r.img, r.img.Bounds(), image.Black, image.Point{}, draw.Src)
}

// DrawText draws text at the specified position
func (r *Renderer) DrawText(x, y int, text string) {
	d := &font.Drawer{
		Dst:  r.img,
		Src:  image.White,
		Face: r.face,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)},
	}
	d.DrawString(text)
}

// DrawTextWrapped draws text with word wrapping
func (r *Renderer) DrawTextWrapped(x, y, maxWidth int, text string) int {
	lineHeight := r.face.Metrics().Height.Ceil()
	currentY := y

	words := splitWords(text)
	line := ""

	for _, word := range words {
		testLine := line
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		// Measure line width
		advance := font.MeasureString(r.face, testLine)
		if advance.Ceil() > maxWidth && line != "" {
			// Draw current line and start new one
			r.DrawText(x, currentY, line)
			currentY += lineHeight
			line = word
		} else {
			line = testLine
		}
	}

	// Draw remaining text
	if line != "" {
		r.DrawText(x, currentY, line)
		currentY += lineHeight
	}

	return currentY - y
}

// DrawRect draws a rectangle outline
func (r *Renderer) DrawRect(x, y, width, height int) {
	for i := x; i < x+width; i++ {
		r.img.SetGray(i, y, color.Gray{Y: 255})
		r.img.SetGray(i, y+height-1, color.Gray{Y: 255})
	}
	for i := y; i < y+height; i++ {
		r.img.SetGray(x, i, color.Gray{Y: 255})
		r.img.SetGray(x+width-1, i, color.Gray{Y: 255})
	}
}

// FillRect draws a filled rectangle
func (r *Renderer) FillRect(x, y, width, height int) {
	for py := y; py < y+height; py++ {
		for px := x; px < x+width; px++ {
			r.img.SetGray(px, py, color.Gray{Y: 255})
		}
	}
}

// SetPixel sets a single pixel
func (r *Renderer) SetPixel(x, y int, on bool) {
	if on {
		r.img.SetGray(x, y, color.Gray{Y: 255})
	} else {
		r.img.SetGray(x, y, color.Gray{Y: 0})
	}
}

// GetFrameBuffer returns the frame buffer as 1-bit packed data
// Format: row-major, 8 pixels per byte, MSB first
func (r *Renderer) GetFrameBuffer() []byte {
	bytesPerRow := (r.width + 7) / 8
	data := make([]byte, bytesPerRow*r.height)

	for y := 0; y < r.height; y++ {
		for x := 0; x < r.width; x++ {
			pixel := r.img.GrayAt(x, y)
			if pixel.Y > 127 { // Threshold to 1-bit
				byteIdx := y*bytesPerRow + x/8
				bitIdx := 7 - (x % 8) // MSB first
				data[byteIdx] |= 1 << bitIdx
			}
		}
	}

	return data
}

// GetRegion returns a portion of the frame buffer
func (r *Renderer) GetRegion(x, y, width, height int) []byte {
	bytesPerRow := (width + 7) / 8
	data := make([]byte, bytesPerRow*height)

	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			pixel := r.img.GrayAt(x+dx, y+dy)
			if pixel.Y > 127 {
				byteIdx := dy*bytesPerRow + dx/8
				bitIdx := 7 - (dx % 8)
				data[byteIdx] |= 1 << bitIdx
			}
		}
	}

	return data
}

// Width returns the renderer width
func (r *Renderer) Width() int {
	return r.width
}

// Height returns the renderer height
func (r *Renderer) Height() int {
	return r.height
}

func splitWords(text string) []string {
	var words []string
	current := ""
	for _, ch := range text {
		if ch == ' ' || ch == '\t' || ch == '\n' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		words = append(words, current)
	}
	return words
}
