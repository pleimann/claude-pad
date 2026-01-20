package pty

import (
	"testing"
)

func TestNewRingBuffer(t *testing.T) {
	rb := NewRingBuffer(10)
	if rb.size != 10 {
		t.Errorf("size = %d, want 10", rb.size)
	}
	if len(rb.data) != 10 {
		t.Errorf("len(data) = %d, want 10", len(rb.data))
	}
}

func TestRingBufferWrite(t *testing.T) {
	rb := NewRingBuffer(5)

	rb.Write([]byte("abc"))
	result := rb.String()

	if result != "abc" {
		t.Errorf("String() = %q, want %q", result, "abc")
	}
}

func TestRingBufferOverwrite(t *testing.T) {
	rb := NewRingBuffer(5)

	// Write more than buffer size
	rb.Write([]byte("hello world"))

	// Should contain last 5 characters
	result := rb.String()
	if result != "world" {
		t.Errorf("String() = %q, want %q", result, "world")
	}
}

func TestRingBufferEmpty(t *testing.T) {
	rb := NewRingBuffer(5)
	result := rb.String()

	if result != "" {
		t.Errorf("String() on empty buffer = %q, want empty", result)
	}
}

func TestRingBufferExactFit(t *testing.T) {
	rb := NewRingBuffer(5)
	rb.Write([]byte("12345"))

	result := rb.String()
	if result != "12345" {
		t.Errorf("String() = %q, want %q", result, "12345")
	}
}

func TestRingBufferMultipleWrites(t *testing.T) {
	rb := NewRingBuffer(10)

	rb.Write([]byte("hello"))
	rb.Write([]byte(" "))
	rb.Write([]byte("world"))

	result := rb.String()
	// 11 chars written to 10-byte buffer, keeps last 10: "ello world"
	if result != "ello world" {
		t.Errorf("String() = %q, want %q", result, "ello world")
	}
}

func TestNewManagerValidation(t *testing.T) {
	// Empty command should fail
	_, err := NewManager("", nil, "")
	if err == nil {
		t.Error("NewManager() with empty command should return error")
	}

	// Valid command should succeed
	m, err := NewManager("echo", []string{"test"}, "")
	if err != nil {
		t.Errorf("NewManager() error = %v", err)
	}
	if m == nil {
		t.Error("NewManager() returned nil")
	}
}

func TestManagerIsRunningBeforeStart(t *testing.T) {
	m, _ := NewManager("echo", []string{"test"}, "")

	if m.IsRunning() {
		t.Error("IsRunning() = true before Start(), want false")
	}
}

func TestManagerGetRecentOutputEmpty(t *testing.T) {
	m, _ := NewManager("echo", []string{"test"}, "")

	output := m.GetRecentOutput()
	if output != "" {
		t.Errorf("GetRecentOutput() = %q, want empty", output)
	}
}
