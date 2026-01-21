package configpush

import (
	"strings"
	"testing"

	"github.com/pleimann/camel-pad/internal/config"
)

func TestGeneratePythonConfig(t *testing.T) {
	cfg := &config.Config{
		Timing: config.TimingConfig{
			DoublePressWindowMs:  300,
			LongPressThresholdMs: 500,
		},
		Buttons: []config.Button{
			{
				Index: 0,
				Name:  "btn_a",
				Press: &config.KeyAction{
					Keys: []string{"ctrl+c"},
				},
				DoublePress: &config.KeyAction{
					Keys: []string{"ctrl+z"},
				},
				LongPress: &config.KeyAction{
					Keys: []string{"q", "enter"},
				},
			},
		},
	}

	content, err := GeneratePythonConfig(cfg)
	if err != nil {
		t.Fatalf("GeneratePythonConfig failed: %v", err)
	}

	// Check that expected content is present
	expectedParts := []string{
		"from adafruit_hid.keycode import Keycode",
		"TIMING = {",
		`"double_press_window_ms": 300`,
		`"long_press_threshold_ms": 500`,
		"BUTTONS = {",
		"0: {  # btn_a",
		`"press": [Keycode.CONTROL, Keycode.C]`,
		`"double_press": [Keycode.CONTROL, Keycode.Z]`,
		`"long_press": [[Keycode.Q], [Keycode.ENTER]]`,
	}

	for _, part := range expectedParts {
		if !strings.Contains(content, part) {
			t.Errorf("Expected config to contain %q, but it didn't.\n\nGenerated config:\n%s", part, content)
		}
	}
}

func TestParseKeyToKeycodes(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
		wantErr  bool
	}{
		{"a", []string{"A"}, false},
		{"enter", []string{"ENTER"}, false},
		{"ctrl+c", []string{"CONTROL", "C"}, false},
		{"ctrl+shift+z", []string{"CONTROL", "SHIFT", "Z"}, false},
		{"alt+f4", []string{"ALT", "F4"}, false},
		{"`", []string{"GRAVE_ACCENT"}, false},
		{"1", []string{"ONE"}, false},
		{"invalid_key", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseKeyToKeycodes(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseKeyToKeycodes(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(result) != len(tt.expected) {
					t.Errorf("ParseKeyToKeycodes(%q) = %v, want %v", tt.input, result, tt.expected)
					return
				}
				for i, v := range result {
					if v != tt.expected[i] {
						t.Errorf("ParseKeyToKeycodes(%q)[%d] = %v, want %v", tt.input, i, v, tt.expected[i])
					}
				}
			}
		})
	}
}
