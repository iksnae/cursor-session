package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/iksnae/cursor-session/internal"
	"github.com/iksnae/cursor-session/internal/export"
	"github.com/spf13/cobra"
)

var (
	format       string
	outputDir    string
	workspace    string
	sessionID    string
	intermediary bool
	clearCache   bool
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export sessions to file",
	Long: `Export chat sessions to various formats (jsonl, md, yaml, json).

You can export all sessions, filter by workspace, or export a specific session by ID.
Use 'cursor-session list' to see available session IDs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get paths (with optional custom storage location)
		paths, err := internal.GetStoragePaths(storagePath)
		if err != nil {
			return fmt.Errorf("failed to get storage paths: %w", err)
		}

		// Copy database files to temp location if --copy flag is set
		var cleanup func() error
		if copyDB {
			var copyErr error
			paths, cleanup, copyErr = internal.CopyStoragePaths(paths)
			if copyErr != nil {
				return fmt.Errorf("failed to copy database files: %w", copyErr)
			}
			// Schedule cleanup when command completes
			defer func() {
				if cleanup != nil {
					if err := cleanup(); err != nil {
						internal.LogWarn("Failed to cleanup temporary files: %v", err)
					} else {
						internal.LogInfo("Cleaned up temporary database files")
					}
				}
			}()
		}

		// Create storage backend (handles both desktop app and agent storage)
		backend, err := internal.NewStorageBackend(paths)
		if err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}

		// Initialize cache manager (always enabled)
		// Store cache in user's home directory root
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		cacheDir := filepath.Join(homeDir, ".cursor-session-cache")
		cacheManager := internal.NewCacheManager(cacheDir)

		// Clear cache if requested
		if clearCache {
			if err := cacheManager.ClearCache(); err != nil {
				internal.LogWarn("Failed to clear cache: %v", err)
			} else {
				internal.LogInfo("Cache cleared")
			}
		}

		var sessions []*internal.Session

		// Use appropriate cache key based on storage type
		var cacheKey string
		if paths.GlobalStorageExists() {
			cacheKey = paths.GetGlobalStorageDBPath()
		} else if paths.HasAgentStorage() {
			cacheKey = paths.AgentStoragePath
		} else {
			cacheKey = "unknown"
		}

		// Try to load from cache
		valid, err := cacheManager.IsCacheValid(cacheKey)
		if err == nil && valid {
			internal.LogInfo("Loading sessions from cache...")
			sessions, err = cacheManager.LoadAllSessions()
			if err == nil && len(sessions) > 0 {
				internal.LogInfo("Loaded %d session(s) from cache", len(sessions))
			} else {
				internal.LogWarn("Failed to load cache: %v, reconstructing...", err)
				sessions = nil
			}
		}

		// Reconstruct if cache miss
		if sessions == nil {
			var conversations []*internal.ReconstructedConversation

			ctx := context.Background()
			steps := []internal.ProgressStep{
				{
					Message: "Loading data from storage",
					Fn: func() error {
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
					},
				},
				{
					Message: "Processing and normalizing sessions",
					Fn: func() error {
						// Detect workspaces for association
						workspaces, _ := internal.DetectWorkspaces(paths.BasePath)

						// Load contexts for workspace association
						var contexts map[string][]*internal.MessageContext
						contexts, _ = backend.LoadMessageContexts()

						// Normalize with workspace association
						normalizer := internal.NewNormalizer()
						sessions = make([]*internal.Session, 0, len(conversations))
						for _, conv := range conversations {
							// Try to associate with workspace
							assignedWorkspace := workspace
							if assignedWorkspace == "" {
								assignedWorkspace = internal.AssociateComposerWithWorkspace(conv.ComposerID, contexts[conv.ComposerID], workspaces)
							}

							session, err := normalizer.NormalizeConversation(conv, assignedWorkspace)
							if err != nil {
								internal.LogWarn("Failed to normalize conversation %s: %v", conv.ComposerID, err)
								continue
							}
							sessions = append(sessions, session)
						}

						// Deduplicate
						deduplicator := internal.NewDeduplicator()
						sessions = deduplicator.Deduplicate(sessions)
						return nil
					},
				},
				{
					Message: "Caching sessions",
					Fn: func() error {
						// Save to cache
						if err := cacheManager.SaveSessions(sessions, cacheKey); err != nil {
							internal.LogWarn("Failed to save cache: %v", err)
						}
						return nil
					},
				},
			}

			if err := internal.ShowProgressWithSteps(ctx, steps); err != nil {
				return err
			}
		}

		// Filter by workspace if specified
		if workspace != "" {
			filtered := make([]*internal.Session, 0)
			for _, session := range sessions {
				if session.Workspace == workspace {
					filtered = append(filtered, session)
				}
			}
			sessions = filtered
		}

		// Filter by session ID if specified
		if sessionID != "" {
			filtered := make([]*internal.Session, 0)
			for _, session := range sessions {
				if session.ID == sessionID {
					filtered = append(filtered, session)
					break // Only one session should match
				}
			}
			if len(filtered) == 0 {
				return fmt.Errorf("session not found: %s (use 'cursor-session list' to see available sessions)", sessionID)
			}
			sessions = filtered
		}

		// Create exporter
		exporter, err := export.NewExporter(format)
		if err != nil {
			return err
		}

		// Ensure output directory exists
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Export sessions with progress
		ctx := context.Background()
		err = internal.ShowProgress(ctx, fmt.Sprintf("Exporting %d session(s) to %s", len(sessions), outputDir), func() error {
			for _, session := range sessions {
				if session == nil {
					internal.LogWarn("Skipping nil session")
					continue
				}
				filename := fmt.Sprintf("session_%s.%s", session.ID, exporter.Extension())
				filepath := filepath.Join(outputDir, filename)

				file, err := os.Create(filepath)
				if err != nil {
					internal.LogError("Failed to create file %s: %v", filepath, err)
					continue
				}

				if err := exporter.Export(session, file); err != nil {
					_ = file.Close()
					internal.LogError("Failed to export session %s: %v", session.ID, err)
					continue
				}

				if err := file.Close(); err != nil {
					internal.LogWarn("Failed to close file %s: %v", filepath, err)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}

		internal.PrintSuccess(fmt.Sprintf("Export complete: %d session(s) exported to %s", len(sessions), outputDir))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringVarP(&format, "format", "f", "jsonl", "Export format (jsonl, md, yaml, json)")
	exportCmd.Flags().StringVarP(&outputDir, "out", "o", "./exports", "Output directory")
	exportCmd.Flags().StringVar(&workspace, "workspace", "", "Filter by workspace")
	exportCmd.Flags().StringVar(&sessionID, "session-id", "", "Export a specific session by ID")
	exportCmd.Flags().BoolVar(&intermediary, "intermediary", false, "Save intermediary format")
	exportCmd.Flags().BoolVar(&clearCache, "clear-cache", false, "Clear the cache before running")
}
