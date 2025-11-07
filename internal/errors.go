package internal

import "fmt"

// StorageError represents errors accessing storage files
type StorageError struct {
	Path string
	Op   string // "open", "read", "parse"
	Err  error
}

func (e *StorageError) Error() string {
	return fmt.Sprintf("storage error: %s %s: %v", e.Op, e.Path, e.Err)
}

func (e *StorageError) Unwrap() error {
	return e.Err
}

// ParseError represents errors parsing data
type ParseError struct {
	Source string // "globalStorage"
	Key    string // storage key or file path
	Err    error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error [%s] %s: %v", e.Source, e.Key, e.Err)
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

// ReconstructionError represents errors during conversation reconstruction
type ReconstructionError struct {
	ComposerID string
	Err        error
}

func (e *ReconstructionError) Error() string {
	return fmt.Sprintf("reconstruction error [%s]: %v", e.ComposerID, e.Err)
}

func (e *ReconstructionError) Unwrap() error {
	return e.Err
}

// ExportError represents errors during export
type ExportError struct {
	Format string
	Path   string
	Err    error
}

func (e *ExportError) Error() string {
	return fmt.Sprintf("export error [%s] %s: %v", e.Format, e.Path, e.Err)
}

func (e *ExportError) Unwrap() error {
	return e.Err
}
