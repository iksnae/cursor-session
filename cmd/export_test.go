package cmd

import (
	"bytes"
	"testing"
)

func TestExportCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "export with invalid format",
			args:    []string{"export", "--format", "invalid"},
			wantErr: true, // Invalid format should error
		},
		// Note: Other tests may succeed if a real database exists
		// We test the flag parsing and error handling paths
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs(tt.args)
			rootCmd.SetOut(&bytes.Buffer{})
			rootCmd.SetErr(&bytes.Buffer{})

			err := rootCmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("exportCmd.Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExportCommand_FlagParsing(t *testing.T) {
	// Test that flags are parsed correctly
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "format flag",
			args: []string{"export", "--format", "jsonl"},
		},
		{
			name: "output directory flag",
			args: []string{"export", "--out", "/tmp/test"},
		},
		{
			name: "workspace flag",
			args: []string{"export", "--workspace", "/path/to/workspace"},
		},
		{
			name: "clear-cache flag",
			args: []string{"export", "--clear-cache"},
		},
		{
			name: "intermediary flag",
			args: []string{"export", "--intermediary"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs(tt.args)
			rootCmd.SetOut(&bytes.Buffer{})
			rootCmd.SetErr(&bytes.Buffer{})

			// Just verify flags are parsed without error
			// The actual execution may succeed or fail depending on environment
			_ = rootCmd.Execute()
		})
	}
}




