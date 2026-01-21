package pty

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/pleimann/camel-pad/internal/action"
)

// Manager manages a PTY and the TUI process running in it
type Manager struct {
	command    string
	args       []string
	workingDir string

	mu     sync.Mutex
	ptmx   *os.File
	cmd    *exec.Cmd
	output []byte // Buffer for recent output (for status parsing)

	outputMu     sync.RWMutex
	outputBuffer *RingBuffer
}

// RingBuffer is a simple ring buffer for storing recent output
type RingBuffer struct {
	data  []byte
	size  int
	write int
}

// NewRingBuffer creates a new ring buffer with the given size
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		data: make([]byte, size),
		size: size,
	}
}

// Write writes data to the ring buffer
func (rb *RingBuffer) Write(p []byte) {
	for _, b := range p {
		rb.data[rb.write] = b
		rb.write = (rb.write + 1) % rb.size
	}
}

// String returns the buffer contents as a string
func (rb *RingBuffer) String() string {
	// Return from oldest to newest
	result := make([]byte, rb.size)
	for i := 0; i < rb.size; i++ {
		result[i] = rb.data[(rb.write+i)%rb.size]
	}
	// Trim null bytes
	start := 0
	for start < len(result) && result[start] == 0 {
		start++
	}
	return string(result[start:])
}

// NewManager creates a new PTY manager
func NewManager(command string, args []string, workingDir string) (*Manager, error) {
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}

	return &Manager{
		command:      command,
		args:         args,
		workingDir:   workingDir,
		outputBuffer: NewRingBuffer(4096), // Keep last 4KB of output
	}, nil
}

// Start starts the TUI process in a PTY
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd := exec.CommandContext(ctx, m.command, m.args...)
	if m.workingDir != "" {
		cmd.Dir = m.workingDir
	}

	// Set up environment
	cmd.Env = os.Environ()

	// Start command with PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to start PTY: %w", err)
	}

	m.ptmx = ptmx
	m.cmd = cmd

	// Start output reader
	go m.readOutput(ctx)

	// Wait for process in background
	go func() {
		m.cmd.Wait()
	}()

	return nil
}

// Stop stops the TUI process and closes the PTY
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Signal(os.Interrupt)
		m.cmd.Process.Wait()
	}

	if m.ptmx != nil {
		m.ptmx.Close()
		m.ptmx = nil
	}
}

// readOutput reads from the PTY and stores output in the ring buffer
func (m *Manager) readOutput(ctx context.Context) {
	buf := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		m.mu.Lock()
		ptmx := m.ptmx
		m.mu.Unlock()

		if ptmx == nil {
			return
		}

		n, err := ptmx.Read(buf)
		if err != nil {
			if err == io.EOF {
				return
			}
			continue
		}

		if n > 0 {
			m.outputMu.Lock()
			m.outputBuffer.Write(buf[:n])
			m.outputMu.Unlock()
		}
	}
}

// WriteKey writes a key press to the PTY
func (m *Manager) WriteKey(key action.KeyPress) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ptmx == nil {
		return fmt.Errorf("PTY not started")
	}

	data := key.ToBytes()
	if data == nil {
		return fmt.Errorf("could not convert key to bytes")
	}

	_, err := m.ptmx.Write(data)
	return err
}

// WriteString writes a string to the PTY
func (m *Manager) WriteString(s string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ptmx == nil {
		return fmt.Errorf("PTY not started")
	}

	_, err := m.ptmx.WriteString(s)
	return err
}

// GetRecentOutput returns recent output from the TUI
func (m *Manager) GetRecentOutput() string {
	m.outputMu.RLock()
	defer m.outputMu.RUnlock()
	return m.outputBuffer.String()
}

// Resize resizes the PTY window
func (m *Manager) Resize(rows, cols uint16) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ptmx == nil {
		return fmt.Errorf("PTY not started")
	}

	return pty.Setsize(m.ptmx, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
}

// IsRunning returns whether the TUI process is running
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd == nil || m.cmd.Process == nil {
		return false
	}

	// Check if process has exited
	return m.cmd.ProcessState == nil || !m.cmd.ProcessState.Exited()
}
