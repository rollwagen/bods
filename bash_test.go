package main

import (
	"testing"
	"time"
)

func TestNewBashSession(t *testing.T) {
	bs := NewBashSession()
	if bs.timeout != 120*time.Second {
		t.Errorf("Expected timeout to be 120s, got %v", bs.timeout)
	}
	if bs.outputDely != 200*time.Millisecond {
		t.Errorf("Expected output delay to be 200ms, got %v", bs.outputDely)
	}
	if bs.sentinel != "<<exit>>" {
		t.Errorf("Expected sentinel to be <<exit>>, got %s", bs.sentinel)
	}
}

func TestBashSessionLifecycle(t *testing.T) {
	bs := NewBashSession()

	// Test Start
	err := bs.Start()
	if err != nil {
		t.Fatalf("Failed to start bash session: %v", err)
	}
	if !bs.started {
		t.Error("Session should be marked as started")
	}

	// Test double start
	err = bs.Start()
	if err != nil {
		t.Error("Second start should not return error")
	}

	// Test basic command execution
	output, errOutput, err := bs.Run("echo 'test'")
	if err != nil {
		t.Errorf("Failed to run command: %v", err)
	}
	if output != "test\n" {
		t.Errorf("Expected output 'test\\n', got '%s'", output)
	}
	if errOutput != "" {
		t.Errorf("Expected empty error output, got '%s'", errOutput)
	}

	// Test Stop
	err = bs.Stop()
	if err != nil {
		t.Errorf("Failed to stop bash session: %v", err)
	}
	if bs.started {
		t.Error("Session should be marked as stopped")
	}
}

func TestBashSessionErrors(t *testing.T) {
	bs := NewBashSession()

	// Test running command before start
	_, _, err := bs.Run("echo 'test'")
	if err == nil {
		t.Error("Expected error when running command before start")
	}

	// Start session for further tests
	err = bs.Start()
	if err != nil {
		t.Fatalf("Failed to start bash session: %v", err)
	}

	// Test invalid command
	_, errOutput, err := bs.Run("invalidcommand")
	if err != nil {
		t.Errorf("Expected no error for invalid command, got %v", err)
	}
	if errOutput == "" {
		t.Error("Expected error output for invalid command")
	}

	// Test timeout scenario with custom short timeout
	bs.timeout = 1 * time.Second
	_, _, err = bs.Run("sleep 2")
	if err == nil {
		t.Error("Expected timeout error")
	}

	// Cleanup
	bs.Stop()
}

func TestBashSessionOutput(t *testing.T) {
	bs := NewBashSession()
	err := bs.Start()
	if err != nil {
		t.Fatalf("Failed to start bash session: %v", err)
	}
	defer bs.Stop()

	// Test multi-line output
	output, errOutput, err := bs.Run("echo 'line1'; echo 'line2'")
	if err != nil {
		t.Errorf("Failed to run command: %v", err)
	}
	expected := "line1\nline2\n"
	if output != expected {
		t.Errorf("Expected output '%s', got '%s'", expected, output)
	}
	if errOutput != "" {
		t.Errorf("Expected empty error output, got '%s'", errOutput)
	}

	// Test command with stderr output
	output, errOutput, err = bs.Run("echo 'error' >&2")
	if err != nil {
		t.Errorf("Failed to run command: %v", err)
	}
	if errOutput != "error\n" {
		t.Errorf("Expected error output 'error\\n', got '%s'", errOutput)
	}
}

func TestBashSessionStopWithoutStart(t *testing.T) {
	bs := NewBashSession()
	err := bs.Stop()
	if err != nil {
		t.Errorf("Stop without start should not return error, got %v", err)
	}
}
