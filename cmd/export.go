package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/k/cursor-session/internal"
	"github.com/k/cursor-session/internal/export"
	"github.com/spf13/cobra"
)

var (
	format       string
	outputDir    string
	workspace    string
	intermediary bool
	clearCache   bool
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export sessions to file",
	Long:  `Export chat sessions to various formats (jsonl, md, yaml, json).`,
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
		dbPath := paths.GetGlobalStorageDBPath()

		// Try to load from cache
		valid, err := cacheManager.IsCacheValid(dbPath)
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
			// Open database
			db, err := internal.OpenDatabase(dbPath)
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

			// Detect workspaces for association
			workspaces, _ := internal.DetectWorkspaces(paths.BasePath)

			// Load contexts for workspace association
			var contexts map[string][]*internal.MessageContext
			contexts, _ = storage.LoadMessageContexts()

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

			// Save to cache
			if err := cacheManager.SaveSessions(sessions, dbPath); err != nil {
				internal.LogWarn("Failed to save cache: %v", err)
			} else {
				internal.LogInfo("Cached %d session(s)", len(sessions))
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

		// Create exporter
		exporter, err := export.NewExporter(format)
		if err != nil {
			return err
		}

		// Ensure output directory exists
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Export sessions
		internal.LogInfo("Exporting %d session(s) to %s", len(sessions), outputDir)
		for i, session := range sessions {
			filename := fmt.Sprintf("session_%s.%s", session.ID, exporter.Extension())
			filepath := filepath.Join(outputDir, filename)

			file, err := os.Create(filepath)
			if err != nil {
				internal.LogError("Failed to create file %s: %v", filepath, err)
				continue
			}

			if err := exporter.Export(session, file); err != nil {
				file.Close()
				internal.LogError("Failed to export session %s: %v", session.ID, err)
				continue
			}

			file.Close()
			internal.LogInfo("Exported session %d/%d: %s", i+1, len(sessions), filepath)
		}

		internal.LogInfo("Export complete: %d session(s) exported", len(sessions))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringVarP(&format, "format", "f", "jsonl", "Export format (jsonl, md, yaml, json)")
	exportCmd.Flags().StringVarP(&outputDir, "out", "o", "./exports", "Output directory")
	exportCmd.Flags().StringVar(&workspace, "workspace", "", "Filter by workspace")
	exportCmd.Flags().BoolVar(&intermediary, "intermediary", false, "Save intermediary format")
	exportCmd.Flags().BoolVar(&clearCache, "clear-cache", false, "Clear the cache before running")
}
