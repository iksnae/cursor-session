package internal

import (
	"errors"
	"strings"
	"testing"
)

func TestStorageError(t *testing.T) {
	originalErr := errors.New("permission denied")
	err := &StorageError{
		Path: "/test/path",
		Op:   "open",
		Err:  originalErr,
	}

	// Test Error() method
	errorMsg := err.Error()
	if errorMsg == "" {
		t.Error("StorageError.Error() returned empty string")
	}
	if !strings.Contains(errorMsg, "storage error") {
		t.Errorf("StorageError.Error() should contain 'storage error', got: %q", errorMsg)
	}
	if !strings.Contains(errorMsg, "/test/path") {
		t.Errorf("StorageError.Error() should contain path, got: %q", errorMsg)
	}

	// Test Unwrap() method
	if !errors.Is(err, originalErr) {
		t.Error("StorageError.Unwrap() should return original error")
	}
}

func TestParseError(t *testing.T) {
	originalErr := errors.New("invalid JSON")
	err := &ParseError{
		Source: "globalStorage",
		Key:    "test:key",
		Err:    originalErr,
	}

	// Test Error() method
	errorMsg := err.Error()
	if errorMsg == "" {
		t.Error("ParseError.Error() returned empty string")
	}
	if !strings.Contains(errorMsg, "parse error") {
		t.Errorf("ParseError.Error() should contain 'parse error', got: %q", errorMsg)
	}
	if !strings.Contains(errorMsg, "globalStorage") {
		t.Errorf("ParseError.Error() should contain source, got: %q", errorMsg)
	}

	// Test Unwrap() method
	if !errors.Is(err, originalErr) {
		t.Error("ParseError.Unwrap() should return original error")
	}
}

func TestReconstructionError(t *testing.T) {
	originalErr := errors.New("bubble not found")
	err := &ReconstructionError{
		ComposerID: "composer1",
		Err:        originalErr,
	}

	// Test Error() method
	errorMsg := err.Error()
	if errorMsg == "" {
		t.Error("ReconstructionError.Error() returned empty string")
	}
	if !strings.Contains(errorMsg, "reconstruction error") {
		t.Errorf("ReconstructionError.Error() should contain 'reconstruction error', got: %q", errorMsg)
	}
	if !strings.Contains(errorMsg, "composer1") {
		t.Errorf("ReconstructionError.Error() should contain ComposerID, got: %q", errorMsg)
	}

	// Test Unwrap() method
	if !errors.Is(err, originalErr) {
		t.Error("ReconstructionError.Unwrap() should return original error")
	}
}

func TestExportError(t *testing.T) {
	originalErr := errors.New("write failed")
	err := &ExportError{
		Format: "jsonl",
		Path:   "/output/file.jsonl",
		Err:    originalErr,
	}

	// Test Error() method
	errorMsg := err.Error()
	if errorMsg == "" {
		t.Error("ExportError.Error() returned empty string")
	}
	if !strings.Contains(errorMsg, "export error") {
		t.Errorf("ExportError.Error() should contain 'export error', got: %q", errorMsg)
	}
	if !strings.Contains(errorMsg, "jsonl") {
		t.Errorf("ExportError.Error() should contain format, got: %q", errorMsg)
	}

	// Test Unwrap() method
	if !errors.Is(err, originalErr) {
		t.Error("ExportError.Unwrap() should return original error")
	}
}
