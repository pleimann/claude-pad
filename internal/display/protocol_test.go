package display

import (
	"testing"

	"github.com/mike/claude-pad/internal/hid"
)

func TestFrameEncoderEncodeFullFrame(t *testing.T) {
	encoder := NewFrameEncoder(128, 64)

	data := []byte{0xAA, 0xBB, 0xCC}
	frame := encoder.EncodeFullFrame(data)

	if frame.Command != hid.DisplayCmdFullFrame {
		t.Errorf("Command = 0x%02X, want 0x%02X", frame.Command, hid.DisplayCmdFullFrame)
	}
	if frame.X != 0 || frame.Y != 0 {
		t.Errorf("Position = (%d, %d), want (0, 0)", frame.X, frame.Y)
	}
	if frame.Width != 128 || frame.Height != 64 {
		t.Errorf("Size = (%d, %d), want (128, 64)", frame.Width, frame.Height)
	}
	if len(frame.Data) != 3 {
		t.Errorf("len(Data) = %d, want 3", len(frame.Data))
	}
}

func TestFrameEncoderEncodePartialFrame(t *testing.T) {
	encoder := NewFrameEncoder(128, 64)

	data := []byte{0x11, 0x22}
	frame := encoder.EncodePartialFrame(10, 20, 32, 16, data)

	if frame.Command != hid.DisplayCmdPartial {
		t.Errorf("Command = 0x%02X, want 0x%02X", frame.Command, hid.DisplayCmdPartial)
	}
	if frame.X != 10 || frame.Y != 20 {
		t.Errorf("Position = (%d, %d), want (10, 20)", frame.X, frame.Y)
	}
	if frame.Width != 32 || frame.Height != 16 {
		t.Errorf("Size = (%d, %d), want (32, 16)", frame.Width, frame.Height)
	}
}

func TestFrameEncoderEncodeClear(t *testing.T) {
	encoder := NewFrameEncoder(128, 64)
	frame := encoder.EncodeClear()

	if frame.Command != hid.DisplayCmdClear {
		t.Errorf("Command = 0x%02X, want 0x%02X", frame.Command, hid.DisplayCmdClear)
	}
}

func TestFrameEncoderMaxPayloadSize(t *testing.T) {
	encoder := NewFrameEncoder(128, 64)
	maxSize := encoder.MaxPayloadSize()

	// Should be 64 - 10 = 54
	if maxSize != 54 {
		t.Errorf("MaxPayloadSize() = %d, want 54", maxSize)
	}
}

func TestFrameEncoderChunkFrame(t *testing.T) {
	// 16 pixels wide = 2 bytes per row
	// 32 rows total
	// Max payload = 54 bytes = 27 rows per chunk
	encoder := NewFrameEncoder(16, 32)

	bytesPerRow := 2
	totalRows := 32
	data := make([]byte, bytesPerRow*totalRows) // 64 bytes

	frames := encoder.ChunkFrame(data)

	// With 54 byte max payload and 2 bytes per row = 27 rows per chunk
	// 32 rows total means: chunk 1 = 27 rows, chunk 2 = 5 rows
	if len(frames) != 2 {
		t.Fatalf("len(frames) = %d, want 2", len(frames))
	}

	// First chunk
	if frames[0].Y != 0 {
		t.Errorf("frame[0].Y = %d, want 0", frames[0].Y)
	}
	if frames[0].Height != 27 {
		t.Errorf("frame[0].Height = %d, want 27", frames[0].Height)
	}
	if len(frames[0].Data) != 27*bytesPerRow {
		t.Errorf("len(frame[0].Data) = %d, want %d", len(frames[0].Data), 27*bytesPerRow)
	}

	// Second chunk
	if frames[1].Y != 27 {
		t.Errorf("frame[1].Y = %d, want 27", frames[1].Y)
	}
	if frames[1].Height != 5 {
		t.Errorf("frame[1].Height = %d, want 5", frames[1].Height)
	}
	if len(frames[1].Data) != 5*bytesPerRow {
		t.Errorf("len(frame[1].Data) = %d, want %d", len(frames[1].Data), 5*bytesPerRow)
	}
}

func TestFrameEncoderChunkFrameSmall(t *testing.T) {
	// Frame that fits in one chunk
	encoder := NewFrameEncoder(8, 8)

	data := make([]byte, 8) // 8 rows * 1 byte per row
	frames := encoder.ChunkFrame(data)

	if len(frames) != 1 {
		t.Fatalf("len(frames) = %d, want 1", len(frames))
	}

	if frames[0].Y != 0 {
		t.Errorf("frame[0].Y = %d, want 0", frames[0].Y)
	}
	if frames[0].Height != 8 {
		t.Errorf("frame[0].Height = %d, want 8", frames[0].Height)
	}
}

func TestFrameEncoderChunkFrameWidthCalculation(t *testing.T) {
	// Test with width that's not a multiple of 8
	// 12 pixels = 2 bytes per row (ceil(12/8) = 2)
	encoder := NewFrameEncoder(12, 4)

	data := make([]byte, 2*4) // 4 rows * 2 bytes per row
	frames := encoder.ChunkFrame(data)

	if len(frames) != 1 {
		t.Fatalf("len(frames) = %d, want 1", len(frames))
	}

	if frames[0].Width != 12 {
		t.Errorf("frame[0].Width = %d, want 12", frames[0].Width)
	}
}
