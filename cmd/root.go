package cmd

import (
	"fmt"
	"os"

	"github.com/iksnae/cursor-session/internal"
	"github.com/spf13/cobra"
)

var (
	verbose     bool
	storagePath string
	copyDB      bool
	version     string = "dev"
	commit      string = "unknown"
	date        string = "unknown"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cursor-session",
	Short: "Extract and export Cursor IDE chat sessions",
	Long: `A powerful CLI tool to extract and export chat sessions from Cursor IDE.

This tool extracts chat sessions from Cursor's modern globalStorage format
and exports them in various formats (JSONL, Markdown, YAML, JSON).

Features:
  • List all your chat sessions with metadata
  • View individual conversations with full context
  • Export in multiple formats (JSONL, Markdown, YAML, JSON)
  • Workspace-aware session organization
  • Intelligent caching for fast access
  • Rich content extraction (code blocks, tool calls, context)

Quick Start:
  cursor-session list                    # List all sessions
  cursor-session show <session-id>        # View a specific session
  cursor-session export --format md      # Export as Markdown

For detailed usage, see: https://github.com/iksnae/cursor-session`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		internal.SetVerbose(verbose)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringVar(&storagePath, "storage", "", "Custom storage location (path to database file or storage directory)")
	rootCmd.PersistentFlags().BoolVar(&copyDB, "copy", false, "Copy database files to temporary location to avoid locking issues")
	
	// Set version template to ensure --version flag works
	rootCmd.SetVersionTemplate(`{{printf "%s\n" .Version}}`)
}
