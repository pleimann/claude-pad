package display

import (
	"github.com/pleimann/camel-pad/internal/hid"
)

// FrameEncoder encodes rendered frames for transmission to the device
type FrameEncoder struct {
	width  int
	height int
}

// NewFrameEncoder creates a new frame encoder
func NewFrameEncoder(width, height int) *FrameEncoder {
	return &FrameEncoder{
		width:  width,
		height: height,
	}
}

// EncodeFullFrame creates a full frame display command
func (e *FrameEncoder) EncodeFullFrame(data []byte) *hid.DisplayFrame {
	return hid.NewFullFrame(uint16(e.width), uint16(e.height), data)
}

// EncodePartialFrame creates a partial frame display command
func (e *FrameEncoder) EncodePartialFrame(x, y, width, height int, data []byte) *hid.DisplayFrame {
	return hid.NewPartialFrame(
		uint16(x), uint16(y),
		uint16(width), uint16(height),
		data,
	)
}

// EncodeClear creates a display clear command
func (e *FrameEncoder) EncodeClear() *hid.DisplayFrame {
	return hid.NewClearCommand()
}

// MaxPayloadSize returns the maximum data payload size for a single HID report
// This is typically 64 bytes minus the header (10 bytes)
func (e *FrameEncoder) MaxPayloadSize() int {
	return 54 // 64 - 10 byte header
}

// ChunkFrame splits a large frame into multiple smaller frames
// that fit within HID report size limits
func (e *FrameEncoder) ChunkFrame(data []byte) []*hid.DisplayFrame {
	bytesPerRow := (e.width + 7) / 8
	maxPayload := e.MaxPayloadSize()
	rowsPerChunk := maxPayload / bytesPerRow

	if rowsPerChunk == 0 {
		rowsPerChunk = 1
	}

	var frames []*hid.DisplayFrame
	y := 0

	for y < e.height {
		chunkHeight := rowsPerChunk
		if y+chunkHeight > e.height {
			chunkHeight = e.height - y
		}

		startByte := y * bytesPerRow
		endByte := (y + chunkHeight) * bytesPerRow
		if endByte > len(data) {
			endByte = len(data)
		}

		chunkData := data[startByte:endByte]
		frame := hid.NewPartialFrame(0, uint16(y), uint16(e.width), uint16(chunkHeight), chunkData)
		frames = append(frames, frame)

		y += chunkHeight
	}

	return frames
}
