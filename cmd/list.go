package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/iksnae/cursor-session/internal"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var (
	listClearCache bool
)

var (
	// Styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62")).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212"))

	idStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)

	countStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	dateStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	workspaceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("135")).
			Italic(true)
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available sessions",
	Long:  `List all available chat sessions from Cursor's globalStorage.`,
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
		if listClearCache {
			if err := cacheManager.ClearCache(); err != nil {
				internal.LogWarn("Failed to clear cache: %v", err)
			} else {
				internal.LogInfo("Cache cleared")
			}
		}

		// Use appropriate cache key based on storage type
		var cacheKey string
		if paths.GlobalStorageExists() {
			cacheKey = paths.GetGlobalStorageDBPath()
		} else if paths.HasAgentStorage() {
			// Use agent storage path as cache key
			cacheKey = paths.AgentStoragePath
		} else {
			cacheKey = "unknown"
		}

		// Try to load from cache
		valid, err := cacheManager.IsCacheValid(cacheKey)
		var index *internal.SessionIndex
		if err == nil && valid {
			internal.LogInfo("Loading from cache...")
			index, err = cacheManager.LoadIndex()
			if err == nil && index != nil {
				internal.LogInfo("Loaded %d session(s) from cache", len(index.Sessions))
			} else {
				internal.LogWarn("Failed to load cache: %v, loading from storage...", err)
				index = nil
			}
		}

		// Load from storage if cache miss
		if index == nil {
			// Load composers
			composers, err := backend.LoadComposers()
			if err != nil {
				return fmt.Errorf("failed to load composers: %w", err)
			}

			// Display sessions from storage
			displaySessionsFromComposers(composers)
			return nil
		}

		// Display sessions from cache index
		displaySessionsFromIndex(index)
		return nil
	},
}

func displaySessionsFromComposers(composers []*internal.RawComposer) {
	if len(composers) == 0 {
		fmt.Println(headerStyle.Render("📋 No sessions found"))
		return
	}

	header := headerStyle.Render(fmt.Sprintf("📋 Found %d session(s)", len(composers)))
	fmt.Println(header)
	fmt.Println()

	// Use tabwriter for aligned columns with better spacing
	w := tabwriter.NewWriter(lipgloss.DefaultRenderer().Output(), 0, 0, 3, ' ', tabwriter.AlignRight)

	// Header row - cleaner format
	_, _ = fmt.Fprintln(w, titleStyle.Render("ID")+"\t"+titleStyle.Render("Name")+"\t"+titleStyle.Render("Messages")+"\t"+titleStyle.Render("Created")+"\t")
	_, _ = fmt.Fprintln(w, strings.Repeat("─", 100))

	for _, composer := range composers {
		name := composer.Name
		if name == "" {
			name = "Untitled"
		}

		// Truncate long names but keep them readable
		if len(name) > 50 {
			name = name[:47] + "..."
		}
		nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
		name = nameStyle.Render(name)

		msgCount := "0"
		if len(composer.FullConversationHeadersOnly) > 0 {
			msgCount = countStyle.Render(strconv.Itoa(len(composer.FullConversationHeadersOnly)))
		}

		created := ""
		if composer.CreatedAt > 0 {
			t := composer.GetCreatedAt()
			now := time.Now()
			diff := now.Sub(t)
			if diff < 24*time.Hour {
				created = dateStyle.Render(t.Format("Today 15:04"))
			} else if diff < 7*24*time.Hour {
				created = dateStyle.Render(t.Format("Mon 15:04"))
			} else if diff < 365*24*time.Hour {
				created = dateStyle.Render(t.Format("Jan 02 15:04"))
			} else {
				created = dateStyle.Render(t.Format("2006-01-02"))
			}
		} else {
			created = dateStyle.Render("—")
		}

		// Show short ID (first 8 chars) for readability, but it's the full composerId
		shortID := composer.ComposerID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		id := idStyle.Render(shortID)

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", id, name, msgCount, created)
	}

	_ = w.Flush()
	fmt.Println()
	if len(composers) > 0 {
		fmt.Println(idStyle.Render("💡 Tip: Use the full ID (e.g., ") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Render(composers[0].ComposerID) +
			idStyle.Render(") with `cursor-session show <id>`"))
	}
}

func displaySessionsFromIndex(index *internal.SessionIndex) {
	if len(index.Sessions) == 0 {
		fmt.Println(headerStyle.Render("📋 No sessions found"))
		return
	}

	header := headerStyle.Render(fmt.Sprintf("📋 Found %d session(s)", len(index.Sessions)))
	fmt.Println(header)
	fmt.Println()

	// Use tabwriter for aligned columns with better spacing
	w := tabwriter.NewWriter(lipgloss.DefaultRenderer().Output(), 0, 0, 3, ' ', tabwriter.AlignRight)

	// Header row - cleaner format
	_, _ = fmt.Fprintln(w, titleStyle.Render("ID")+"\t"+titleStyle.Render("Name")+"\t"+titleStyle.Render("Messages")+"\t"+titleStyle.Render("Created")+"\t"+titleStyle.Render("Workspace")+"\t")
	_, _ = fmt.Fprintln(w, strings.Repeat("─", 120))

	for _, entry := range index.Sessions {
		name := entry.Name
		if name == "" {
			name = "Untitled"
		}

		// Truncate long names but keep them readable
		if len(name) > 50 {
			name = name[:47] + "..."
		}
		nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
		name = nameStyle.Render(name)

		msgCount := countStyle.Render(strconv.Itoa(entry.MessageCount))

		created := ""
		if entry.CreatedAt != "" {
			// Parse and format date
			if t, err := time.Parse(time.RFC3339, entry.CreatedAt); err == nil {
				now := time.Now()
				diff := now.Sub(t)
				if diff < 24*time.Hour {
					created = dateStyle.Render(t.Format("Today 15:04"))
				} else if diff < 7*24*time.Hour {
					created = dateStyle.Render(t.Format("Mon 15:04"))
				} else if diff < 365*24*time.Hour {
					created = dateStyle.Render(t.Format("Jan 02 15:04"))
				} else {
					created = dateStyle.Render(t.Format("2006-01-02"))
				}
			} else {
				created = dateStyle.Render(entry.CreatedAt[:10])
			}
		} else {
			created = dateStyle.Render("—")
		}

		workspace := ""
		if entry.Workspace != "" {
			// Truncate workspace path for display but keep it readable
			workspacePath := entry.Workspace
			// Extract just the folder name if it's a full path
			if strings.Contains(workspacePath, "/") {
				parts := strings.Split(workspacePath, "/")
				workspacePath = parts[len(parts)-1]
			}
			if len(workspacePath) > 25 {
				workspacePath = workspacePath[:22] + "..."
			}
			workspace = workspaceStyle.Render(workspacePath)
		} else {
			workspace = dateStyle.Render("—")
		}

		// Show short ID (first 8 chars) for readability
		shortID := entry.ComposerID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		id := idStyle.Render(shortID)

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t\n", id, name, msgCount, created, workspace)
	}

	_ = w.Flush()
	fmt.Println()
	if len(index.Sessions) > 0 {
		fmt.Println(idStyle.Render("💡 Tip: Use the full ID (e.g., ") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Render(index.Sessions[0].ComposerID) +
			idStyle.Render(") with `cursor-session show <id>`"))
	}
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&listClearCache, "clear-cache", false, "Clear the cache before running")
}
