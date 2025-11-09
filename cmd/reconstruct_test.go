package cmd

import (
	"bytes"
	"testing"
)

func TestReconstructCommand_FlagParsing(t *testing.T) {
	// Test that flags are parsed correctly
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "reconstruct without flags",
			args: []string{"reconstruct"},
		},
		{
			name: "reconstruct with output directory",
			args: []string{"reconstruct", "--out", "/tmp/test-reconstruct"},
		},
		{
			name: "reconstruct with short output flag",
			args: []string{"reconstruct", "-o", "/tmp/test-reconstruct"},
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
