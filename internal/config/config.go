package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Device  DeviceConfig  `yaml:"device"`
	Timing  TimingConfig  `yaml:"timing"`
	TUI     TUIConfig     `yaml:"tui"`
	Buttons []Button      `yaml:"buttons"`
	Chords  []Chord       `yaml:"chords"`
	Display DisplayConfig `yaml:"display"`
}

type DeviceConfig struct {
	VendorID       uint16 `yaml:"vendor_id"`
	ProductID      uint16 `yaml:"product_id"`
	PollIntervalMs int    `yaml:"poll_interval_ms"`
}

type TimingConfig struct {
	DoublePressWindowMs  int `yaml:"double_press_window_ms"`
	LongPressThresholdMs int `yaml:"long_press_threshold_ms"`
	ChordWindowMs        int `yaml:"chord_window_ms"`
}

type TUIConfig struct {
	Command    string   `yaml:"command"`
	Args       []string `yaml:"args"`
	WorkingDir string   `yaml:"working_dir,omitempty"`
}

type Button struct {
	Index       int        `yaml:"index"`
	Name        string     `yaml:"name,omitempty"`
	Press       *KeyAction `yaml:"press,omitempty"`
	DoublePress *KeyAction `yaml:"double_press,omitempty"`
	LongPress   *KeyAction `yaml:"long_press,omitempty"`
}

type KeyAction struct {
	Keys []string `yaml:"keys"`
}

type Chord struct {
	Buttons []int    `yaml:"buttons"`
	Keys    []string `yaml:"keys"`
}

type DisplayConfig struct {
	Width            int             `yaml:"width"`
	Height           int             `yaml:"height"`
	UpdateIntervalMs int             `yaml:"update_interval_ms"`
	Regions          []DisplayRegion `yaml:"regions,omitempty"`
}

type DisplayRegion struct {
	Name    string `yaml:"name"`
	X       int    `yaml:"x"`
	Y       int    `yaml:"y"`
	Width   int    `yaml:"width"`
	Height  int    `yaml:"height"`
	Source  string `yaml:"source"`
	Content string `yaml:"content,omitempty"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	cfg.applyDefaults()

	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Device.VendorID == 0 {
		return fmt.Errorf("device.vendor_id is required")
	}
	if c.Device.ProductID == 0 {
		return fmt.Errorf("device.product_id is required")
	}
	if c.TUI.Command == "" {
		return fmt.Errorf("tui.command is required")
	}

	// Validate button indices are unique
	seen := make(map[int]bool)
	for _, btn := range c.Buttons {
		if seen[btn.Index] {
			return fmt.Errorf("duplicate button index: %d", btn.Index)
		}
		seen[btn.Index] = true
	}

	// Validate chord button references exist
	for i, chord := range c.Chords {
		if len(chord.Buttons) < 2 {
			return fmt.Errorf("chord %d must have at least 2 buttons", i)
		}
	}

	return nil
}

func (c *Config) applyDefaults() {
	if c.Device.PollIntervalMs == 0 {
		c.Device.PollIntervalMs = 10
	}
	if c.Timing.DoublePressWindowMs == 0 {
		c.Timing.DoublePressWindowMs = 300
	}
	if c.Timing.LongPressThresholdMs == 0 {
		c.Timing.LongPressThresholdMs = 500
	}
	if c.Timing.ChordWindowMs == 0 {
		c.Timing.ChordWindowMs = 50
	}
	if c.Display.Width == 0 {
		c.Display.Width = 128
	}
	if c.Display.Height == 0 {
		c.Display.Height = 64
	}
	if c.Display.UpdateIntervalMs == 0 {
		c.Display.UpdateIntervalMs = 100
	}
}

// UpdateDeviceIDs updates the vendor_id and product_id in a config file
// while preserving the rest of the file structure and comments
func UpdateDeviceIDs(path string, vendorID, productID uint16) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	content := string(data)

	// Update vendor_id (YAML format: vendor_id: 0x1234 or vendor_id: 1234)
	vendorRegex := regexp.MustCompile(`(?m)^(\s*vendor_id:\s*)(?:0x[0-9A-Fa-f]+|\d+)`)
	content = vendorRegex.ReplaceAllString(content, fmt.Sprintf("${1}0x%04X", vendorID))

	// Update product_id
	productRegex := regexp.MustCompile(`(?m)^(\s*product_id:\s*)(?:0x[0-9A-Fa-f]+|\d+)`)
	content = productRegex.ReplaceAllString(content, fmt.Sprintf("${1}0x%04X", productID))

	// Write back
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// CreateDefaultConfig creates a new config file with default values and the specified device
func CreateDefaultConfig(path string, vendorID, productID uint16) error {
	content := fmt.Sprintf(`# Claude Pad Configuration

device:
  vendor_id: 0x%04X
  product_id: 0x%04X
  poll_interval_ms: 10

timing:
  double_press_window_ms: 300
  long_press_threshold_ms: 500
  chord_window_ms: 50

tui:
  command: "your-tui-app"
  args: []

# Button mappings
buttons:
  - index: 0
    name: btn_0
    press:
      keys: ["enter"]

display:
  width: 128
  height: 64
  update_interval_ms: 100
`, vendorID, productID)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	return nil
}

// Exists checks if a config file exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
