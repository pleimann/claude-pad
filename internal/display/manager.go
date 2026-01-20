package display

import (
	"context"
	"regexp"
	"sync"
	"time"

	"github.com/mike/claude-pad/internal/config"
	"github.com/mike/claude-pad/internal/hid"
	"github.com/mike/claude-pad/internal/pty"
)

// DeviceWriter is the interface for sending frames to the device
type DeviceWriter interface {
	SendFrame(frame *hid.DisplayFrame) error
}

// Manager manages the OLED display, orchestrating rendering and updates
type Manager struct {
	config   config.DisplayConfig
	device   DeviceWriter
	renderer *Renderer
	encoder  *FrameEncoder

	mu            sync.RWMutex
	regions       map[string]*regionState
	statusPattern *regexp.Regexp
	running       bool
	cancel        context.CancelFunc
}

type regionState struct {
	config  config.DisplayRegion
	content string
	dirty   bool
}

// NewManager creates a new display manager
func NewManager(cfg config.DisplayConfig, device DeviceWriter) *Manager {
	m := &Manager{
		config:   cfg,
		device:   device,
		renderer: NewRenderer(cfg.Width, cfg.Height),
		encoder:  NewFrameEncoder(cfg.Width, cfg.Height),
		regions:  make(map[string]*regionState),
	}

	// Initialize regions from config
	for _, regionCfg := range cfg.Regions {
		m.regions[regionCfg.Name] = &regionState{
			config:  regionCfg,
			content: regionCfg.Content, // Static content from config
			dirty:   true,
		}
	}

	// Compile status parsing pattern (looks for lines like "STATUS: text")
	m.statusPattern = regexp.MustCompile(`(?m)^STATUS:\s*(.+)$`)

	return m
}

// Start starts the display update loop
func (m *Manager) Start(ctx context.Context, ptyMgr *pty.Manager) {
	ctx, m.cancel = context.WithCancel(ctx)
	m.running = true

	interval := time.Duration(m.config.UpdateIntervalMs) * time.Millisecond
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.update(ptyMgr)
			}
		}
	}()
}

// Stop stops the display update loop
func (m *Manager) Stop() {
	m.running = false
	if m.cancel != nil {
		m.cancel()
	}

	// Send clear command
	if m.device != nil {
		m.device.SendFrame(m.encoder.EncodeClear())
	}
}

// SetRegionContent sets the content of a named region
func (m *Manager) SetRegionContent(name, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if region, ok := m.regions[name]; ok {
		if region.content != content {
			region.content = content
			region.dirty = true
		}
	}
}

// update performs a display update cycle
func (m *Manager) update(ptyMgr *pty.Manager) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Parse TUI output for status updates
	if ptyMgr != nil {
		output := ptyMgr.GetRecentOutput()
		m.parseStatus(output)
	}

	// Check if any region needs updating
	needsRender := false
	for _, region := range m.regions {
		if region.dirty {
			needsRender = true
			break
		}
	}

	if !needsRender {
		return
	}

	// Render all regions
	m.renderer.Clear()
	for _, region := range m.regions {
		m.renderRegion(region)
		region.dirty = false
	}

	// Send frame to device
	frameData := m.renderer.GetFrameBuffer()
	frames := m.encoder.ChunkFrame(frameData)

	for _, frame := range frames {
		if err := m.device.SendFrame(frame); err != nil {
			// Log error but continue
			continue
		}
	}
}

// parseStatus extracts status text from TUI output
func (m *Manager) parseStatus(output string) {
	matches := m.statusPattern.FindStringSubmatch(output)
	if len(matches) >= 2 {
		status := matches[1]
		// Update any region with source "tui_status"
		for _, region := range m.regions {
			if region.config.Source == "tui_status" && region.content != status {
				region.content = status
				region.dirty = true
			}
		}
	}
}

// renderRegion renders a single region to the frame buffer
func (m *Manager) renderRegion(region *regionState) {
	cfg := region.config

	switch cfg.Source {
	case "static", "tui_status":
		// Render text content
		// Add padding for text
		textY := cfg.Y + 12 // Account for font baseline
		m.renderer.DrawTextWrapped(cfg.X+2, textY, cfg.Width-4, region.content)

	case "system":
		// Could render system info (time, etc.)
		textY := cfg.Y + 12
		m.renderer.DrawText(cfg.X+2, textY, region.content)
	}
}

// ForceRefresh marks all regions as dirty and triggers an immediate update
func (m *Manager) ForceRefresh() {
	m.mu.Lock()
	for _, region := range m.regions {
		region.dirty = true
	}
	m.mu.Unlock()
}
