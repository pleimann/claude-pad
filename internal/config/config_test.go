package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file in YAML format
	content := `
device:
  vendor_id: 0x1234
  product_id: 0x5678
  poll_interval_ms: 20

timing:
  double_press_window_ms: 250
  long_press_threshold_ms: 400
  chord_window_ms: 40

tui:
  command: "test-app"
  args: ["--flag", "value"]
  working_dir: "/tmp"

buttons:
  - index: 0
    name: btn_a
    press:
      keys: ["ctrl+c"]
    double_press:
      keys: ["ctrl+z"]
    long_press:
      keys: ["q", "enter"]

  - index: 1
    name: btn_b
    press:
      keys: ["down"]

chords:
  - buttons: [0, 1]
    keys: ["ctrl+r"]

display:
  width: 128
  height: 64
  update_interval_ms: 50
  regions:
    - name: status
      x: 0
      y: 0
      width: 128
      height: 32
      source: tui_status
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Check device config
	if cfg.Device.VendorID != 0x1234 {
		t.Errorf("VendorID = 0x%04X, want 0x1234", cfg.Device.VendorID)
	}
	if cfg.Device.ProductID != 0x5678 {
		t.Errorf("ProductID = 0x%04X, want 0x5678", cfg.Device.ProductID)
	}
	if cfg.Device.PollIntervalMs != 20 {
		t.Errorf("PollIntervalMs = %d, want 20", cfg.Device.PollIntervalMs)
	}

	// Check timing config
	if cfg.Timing.DoublePressWindowMs != 250 {
		t.Errorf("DoublePressWindowMs = %d, want 250", cfg.Timing.DoublePressWindowMs)
	}
	if cfg.Timing.LongPressThresholdMs != 400 {
		t.Errorf("LongPressThresholdMs = %d, want 400", cfg.Timing.LongPressThresholdMs)
	}
	if cfg.Timing.ChordWindowMs != 40 {
		t.Errorf("ChordWindowMs = %d, want 40", cfg.Timing.ChordWindowMs)
	}

	// Check TUI config
	if cfg.TUI.Command != "test-app" {
		t.Errorf("Command = %q, want %q", cfg.TUI.Command, "test-app")
	}
	if len(cfg.TUI.Args) != 2 || cfg.TUI.Args[0] != "--flag" {
		t.Errorf("Args = %v, want [--flag value]", cfg.TUI.Args)
	}
	if cfg.TUI.WorkingDir != "/tmp" {
		t.Errorf("WorkingDir = %q, want /tmp", cfg.TUI.WorkingDir)
	}

	// Check buttons
	if len(cfg.Buttons) != 2 {
		t.Fatalf("len(Buttons) = %d, want 2", len(cfg.Buttons))
	}
	btn := cfg.Buttons[0]
	if btn.Index != 0 || btn.Name != "btn_a" {
		t.Errorf("Button[0] = {%d, %s}, want {0, btn_a}", btn.Index, btn.Name)
	}
	if btn.Press == nil || len(btn.Press.Keys) != 1 || btn.Press.Keys[0] != "ctrl+c" {
		t.Errorf("Button[0].Press.Keys = %v, want [ctrl+c]", btn.Press.Keys)
	}
	if btn.DoublePress == nil || btn.DoublePress.Keys[0] != "ctrl+z" {
		t.Errorf("Button[0].DoublePress.Keys = %v, want [ctrl+z]", btn.DoublePress.Keys)
	}
	if btn.LongPress == nil || len(btn.LongPress.Keys) != 2 {
		t.Errorf("Button[0].LongPress.Keys = %v, want [q enter]", btn.LongPress.Keys)
	}

	// Check chords
	if len(cfg.Chords) != 1 {
		t.Fatalf("len(Chords) = %d, want 1", len(cfg.Chords))
	}
	chord := cfg.Chords[0]
	if len(chord.Buttons) != 2 || chord.Buttons[0] != 0 || chord.Buttons[1] != 1 {
		t.Errorf("Chord.Buttons = %v, want [0 1]", chord.Buttons)
	}
	if len(chord.Keys) != 1 || chord.Keys[0] != "ctrl+r" {
		t.Errorf("Chord.Keys = %v, want [ctrl+r]", chord.Keys)
	}

	// Check display
	if cfg.Display.Width != 128 || cfg.Display.Height != 64 {
		t.Errorf("Display size = %dx%d, want 128x64", cfg.Display.Width, cfg.Display.Height)
	}
	if len(cfg.Display.Regions) != 1 {
		t.Fatalf("len(Display.Regions) = %d, want 1", len(cfg.Display.Regions))
	}
}

func TestLoadDefaults(t *testing.T) {
	// Minimal config to test defaults
	content := `
device:
  vendor_id: 0x1234
  product_id: 0x5678

tui:
  command: "test-app"
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Check defaults were applied
	if cfg.Device.PollIntervalMs != 10 {
		t.Errorf("PollIntervalMs = %d, want default 10", cfg.Device.PollIntervalMs)
	}
	if cfg.Timing.DoublePressWindowMs != 300 {
		t.Errorf("DoublePressWindowMs = %d, want default 300", cfg.Timing.DoublePressWindowMs)
	}
	if cfg.Timing.LongPressThresholdMs != 500 {
		t.Errorf("LongPressThresholdMs = %d, want default 500", cfg.Timing.LongPressThresholdMs)
	}
	if cfg.Timing.ChordWindowMs != 50 {
		t.Errorf("ChordWindowMs = %d, want default 50", cfg.Timing.ChordWindowMs)
	}
	if cfg.Display.Width != 128 {
		t.Errorf("Display.Width = %d, want default 128", cfg.Display.Width)
	}
	if cfg.Display.Height != 64 {
		t.Errorf("Display.Height = %d, want default 64", cfg.Display.Height)
	}
	if cfg.Display.UpdateIntervalMs != 100 {
		t.Errorf("Display.UpdateIntervalMs = %d, want default 100", cfg.Display.UpdateIntervalMs)
	}
}

func TestLoadValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name: "missing vendor_id",
			content: `
device:
  product_id: 0x5678
tui:
  command: "test"
`,
			wantErr: "vendor_id is required",
		},
		{
			name: "missing product_id",
			content: `
device:
  vendor_id: 0x1234
tui:
  command: "test"
`,
			wantErr: "product_id is required",
		},
		{
			name: "missing tui command",
			content: `
device:
  vendor_id: 0x1234
  product_id: 0x5678
`,
			wantErr: "command is required",
		},
		{
			name: "duplicate button index",
			content: `
device:
  vendor_id: 0x1234
  product_id: 0x5678
tui:
  command: "test"
buttons:
  - index: 0
  - index: 0
`,
			wantErr: "duplicate button index",
		},
		{
			name: "chord with single button",
			content: `
device:
  vendor_id: 0x1234
  product_id: 0x5678
tui:
  command: "test"
chords:
  - buttons: [0]
    keys: ["enter"]
`,
			wantErr: "must have at least 2 buttons",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write temp config: %v", err)
			}

			_, err := Load(configPath)
			if err == nil {
				t.Fatal("Load() expected error, got nil")
			}
			if tt.wantErr != "" && !contains(err.Error(), tt.wantErr) {
				t.Errorf("Load() error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Load() expected error for nonexistent file, got nil")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestUpdateDeviceIDs(t *testing.T) {
	// Create initial config in YAML format
	content := `# Test config
device:
  vendor_id: 0x1234
  product_id: 0x5678
  poll_interval_ms: 10

tui:
  command: "test"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	// Update device IDs
	if err := UpdateDeviceIDs(configPath, 0xABCD, 0xEF01); err != nil {
		t.Fatalf("UpdateDeviceIDs() error = %v", err)
	}

	// Read and verify
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	result := string(data)
	if !contains(result, "vendor_id: 0xABCD") {
		t.Errorf("vendor_id not updated correctly in: %s", result)
	}
	if !contains(result, "product_id: 0xEF01") {
		t.Errorf("product_id not updated correctly in: %s", result)
	}
	// Verify comment is preserved
	if !contains(result, "# Test config") {
		t.Errorf("comment not preserved in: %s", result)
	}
}

func TestUpdateDeviceIDsDecimal(t *testing.T) {
	// Test updating config with decimal IDs
	content := `device:
  vendor_id: 4660
  product_id: 22136

tui:
  command: "test"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	if err := UpdateDeviceIDs(configPath, 0x1111, 0x2222); err != nil {
		t.Fatalf("UpdateDeviceIDs() error = %v", err)
	}

	data, _ := os.ReadFile(configPath)
	result := string(data)
	if !contains(result, "vendor_id: 0x1111") {
		t.Errorf("vendor_id not updated correctly in: %s", result)
	}
	if !contains(result, "product_id: 0x2222") {
		t.Errorf("product_id not updated correctly in: %s", result)
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "new-config.yaml")

	if err := CreateDefaultConfig(configPath, 0x1234, 0x5678); err != nil {
		t.Fatalf("CreateDefaultConfig() error = %v", err)
	}

	// Verify file exists
	if !Exists(configPath) {
		t.Fatal("Config file was not created")
	}

	// Load and validate
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load created config: %v", err)
	}

	if cfg.Device.VendorID != 0x1234 {
		t.Errorf("VendorID = 0x%04X, want 0x1234", cfg.Device.VendorID)
	}
	if cfg.Device.ProductID != 0x5678 {
		t.Errorf("ProductID = 0x%04X, want 0x5678", cfg.Device.ProductID)
	}
}

func TestExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Non-existent file
	if Exists(filepath.Join(tmpDir, "nonexistent.yaml")) {
		t.Error("Exists() = true for non-existent file")
	}

	// Create a file
	existingPath := filepath.Join(tmpDir, "exists.yaml")
	os.WriteFile(existingPath, []byte("test"), 0644)

	if !Exists(existingPath) {
		t.Error("Exists() = false for existing file")
	}
}
