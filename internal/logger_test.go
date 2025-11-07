package internal

import (
	"testing"
)

func TestSetLogLevel(t *testing.T) {
	originalLevel := logLevel
	defer func() { logLevel = originalLevel }()

	SetLogLevel(LogLevelDebug)
	if logLevel != LogLevelDebug {
		t.Errorf("SetLogLevel() logLevel = %v, want LogLevelDebug", logLevel)
	}

	SetLogLevel(LogLevelError)
	if logLevel != LogLevelError {
		t.Errorf("SetLogLevel() logLevel = %v, want LogLevelError", logLevel)
	}
}

func TestSetVerbose(t *testing.T) {
	originalLevel := logLevel
	defer func() { logLevel = originalLevel }()

	SetVerbose(true)
	if logLevel != LogLevelDebug {
		t.Errorf("SetVerbose(true) logLevel = %v, want LogLevelDebug", logLevel)
	}

	SetVerbose(false)
	if logLevel != LogLevelInfo {
		t.Errorf("SetVerbose(false) logLevel = %v, want LogLevelInfo", logLevel)
	}
}

func TestLogFunctions(t *testing.T) {
	// These functions don't return errors, so we just test they don't panic
	// In a real scenario, you might capture output to verify messages

	LogError("test error message")
	LogWarn("test warning message")
	LogInfo("test info message")
	LogDebug("test debug message")

	// If we get here without panic, the functions work
}

func TestLogLevels(t *testing.T) {
	// Test that log levels are properly defined
	if LogLevelError >= LogLevelWarn {
		t.Error("LogLevelError should be less than LogLevelWarn")
	}
	if LogLevelWarn >= LogLevelInfo {
		t.Error("LogLevelWarn should be less than LogLevelInfo")
	}
	if LogLevelInfo >= LogLevelDebug {
		t.Error("LogLevelInfo should be less than LogLevelDebug")
	}
}


