package cmd

import (
	"fmt"
	"os"

	"github.com/iksnae/cursor-session/internal"
	"github.com/spf13/cobra"
)

var (
	verbose bool
	version string = "dev"
	commit  string = "unknown"
	date    string = "unknown"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cursor-session",
	Short: "Export Cursor Editor chat sessions",
	Long: `A CLI tool to extract and export chat sessions from Cursor Editor.

This tool extracts chat sessions from Cursor's modern globalStorage format
and exports them in various formats (JSONL, Markdown, YAML, JSON).`,
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
}
