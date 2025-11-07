package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/iksnae/cursor-session/internal"
	"github.com/spf13/cobra"
)

var (
	reconstructOutput string
)

// reconstructCmd represents the reconstruct command
var reconstructCmd = &cobra.Command{
	Use:   "reconstruct",
	Short: "Reconstruct and save intermediary format",
	Long:  `Reconstruct conversations and save to intermediary JSON/YAML format for debugging.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Detect paths
		paths, err := internal.DetectStoragePaths()
		if err != nil {
			return fmt.Errorf("failed to detect storage paths: %w", err)
		}

		// Check if globalStorage exists
		if !paths.GlobalStorageExists() {
			return fmt.Errorf("globalStorage not found at %s", paths.GetGlobalStorageDBPath())
		}

		// Open database
		db, err := internal.OpenDatabase(paths.GetGlobalStorageDBPath())
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		// Load data asynchronously
		storage := internal.NewStorage(db)
		bubbleChan, composerChan, contextChan, err := internal.LoadDataAsync(storage)
		if err != nil {
			return fmt.Errorf("failed to load data: %w", err)
		}

		// Reconstruct conversations
		conversations, err := internal.ReconstructAsync(bubbleChan, composerChan, contextChan)
		if err != nil {
			return fmt.Errorf("failed to reconstruct conversations: %w", err)
		}

		// Ensure output directory exists
		if err := os.MkdirAll(reconstructOutput, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Save intermediary format
		internal.LogInfo("Saving %d conversation(s) to intermediary format", len(conversations))
		for i, conv := range conversations {
			filename := fmt.Sprintf("conversation_%s.json", conv.ComposerID)
			filepath := filepath.Join(reconstructOutput, filename)

			data, err := json.MarshalIndent(conv, "", "  ")
			if err != nil {
				internal.LogError("Failed to marshal conversation %s: %v", conv.ComposerID, err)
				continue
			}

			if err := os.WriteFile(filepath, data, 0644); err != nil {
				internal.LogError("Failed to write file %s: %v", filepath, err)
				continue
			}

			internal.LogInfo("Saved conversation %d/%d: %s", i+1, len(conversations), filepath)
		}

		internal.LogInfo("Reconstruction complete: %d conversation(s) saved", len(conversations))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(reconstructCmd)
	reconstructCmd.Flags().StringVarP(&reconstructOutput, "out", "o", "./intermediary", "Output directory for intermediary format")
}
