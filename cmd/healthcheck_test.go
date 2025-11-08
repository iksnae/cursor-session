package cmd

import (
	"bytes"
	"testing"
)

func TestHealthcheckCommand(t *testing.T) {
	// Test that the command exists and can be called
	rootCmd.SetArgs([]string{"healthcheck", "--help"})
	
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("healthcheck command failed: %v", err)
	}
	
	output := buf.String()
	if output == "" {
		t.Error("healthcheck --help should produce output")
	}
}

func TestHealthcheckCommandExists(t *testing.T) {
	// Verify healthcheck command is registered
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "healthcheck" {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("healthcheck command not found in root command")
	}
}

func TestHealthcheckVerboseFlag(t *testing.T) {
	// Test that verbose flag exists
	healthcheckCmd := rootCmd.Commands()[0]
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "healthcheck" {
			healthcheckCmd = cmd
			break
		}
	}
	
	verboseFlag := healthcheckCmd.Flag("verbose")
	if verboseFlag == nil {
		t.Error("healthcheck command should have --verbose flag")
	}
	
	shortFlag := healthcheckCmd.Flag("v")
	if shortFlag == nil {
		t.Error("healthcheck command should have -v flag")
	}
}

