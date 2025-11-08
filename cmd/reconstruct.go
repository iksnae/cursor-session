package cmd

import (
	"context"
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

		// Create storage backend (handles both desktop app and agent storage)
		backend, err := internal.NewStorageBackend(paths)
		if err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}

		var conversations []*internal.ReconstructedConversation

		// Load data asynchronously with progress
		ctx := context.Background()
		err = internal.ShowProgress(ctx, "Loading data from storage", func() error {
			var loadErr error
			bubbleChan, composerChan, contextChan, loadErr := internal.LoadDataAsyncFromBackend(backend)
			if loadErr != nil {
				return fmt.Errorf("failed to load data: %w", loadErr)
			}

			// Reconstruct conversations
			conversations, loadErr = internal.ReconstructAsync(bubbleChan, composerChan, contextChan)
			if loadErr != nil {
				return fmt.Errorf("failed to reconstruct conversations: %w", loadErr)
			}
			return nil
		})
		if err != nil {
			return err
		}

		// Ensure output directory exists
		if err := os.MkdirAll(reconstructOutput, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Save intermediary format with progress
		saveCtx := context.Background()
		err = internal.ShowProgress(saveCtx, fmt.Sprintf("Saving %d conversation(s) to intermediary format", len(conversations)), func() error {
			for _, conv := range conversations {
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
			}
			return nil
		})
		if err != nil {
			return err
		}

		internal.PrintSuccess(fmt.Sprintf("Reconstruction complete: %d conversation(s) saved to %s", len(conversations), reconstructOutput))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(reconstructCmd)
	reconstructCmd.Flags().StringVarP(&reconstructOutput, "out", "o", "./intermediary", "Output directory for intermediary format")
}
