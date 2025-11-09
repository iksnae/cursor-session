package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/iksnae/cursor-session/internal"
	"github.com/spf13/cobra"
)

var (
	limit int
	since string
)

var (
	// Styles for show command
	sessionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("212")).
				Padding(0, 1).
				MarginBottom(1)

	sessionMetaStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				MarginBottom(1)

	userMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Bold(true).
				Padding(0, 1)

	assistantMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("135")).
				Bold(true).
				Padding(0, 1)

	messageContentStyle = lipgloss.NewStyle().
				Padding(0, 2).
				MarginBottom(1)

	timestampStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)
)

// showCmd represents the show command
var showCmd = &cobra.Command{
	Use:   "show <session-id>",
	Short: "Show messages for a specific session",
	Long:  `Display messages from a specific chat session.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]

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

		// Use appropriate cache key based on storage type
		var cacheKey string
		if paths.GlobalStorageExists() {
			cacheKey = paths.GetGlobalStorageDBPath()
		} else if paths.HasAgentStorage() {
			cacheKey = paths.AgentStoragePath
		} else {
			cacheKey = "unknown"
		}

		var session *internal.Session

		// Try to load from cache (even if cache is "invalid", individual sessions may still be valid)
		// First check if cache is valid
		valid, err := cacheManager.IsCacheValid(cacheKey)
		if err != nil {
			internal.LogDebug("Cache validation error: %v", err)
		} else if !valid {
			internal.LogDebug("Cache is invalid or doesn't exist, but checking for individual session...")
		} else {
			internal.LogDebug("Cache is valid")
		}

		// Try to load index and find session (even if cache is invalid)
		index, err := cacheManager.LoadIndex()
		if err == nil && index != nil {
			// Verify index is for the same database (path check)
			if index.Metadata.DatabasePath == cacheKey {
				internal.LogDebug("Index loaded with %d sessions, searching for composer ID: %s", len(index.Sessions), sessionID)
				// Find session by composer ID
				for _, entry := range index.Sessions {
					if entry.ComposerID == sessionID {
						internal.LogDebug("Found matching entry, loading session: %s", entry.ID)
						session, err = cacheManager.LoadSession(entry.ID)
						if err == nil {
							internal.LogInfo("Found session in cache")
							break
						} else {
							internal.LogDebug("Failed to load session file: %v", err)
						}
					}
				}
			} else {
				internal.LogDebug("Index is for different database path, ignoring")
			}
		} else {
			internal.LogDebug("Failed to load index: %v", err)
		}

		// Load from storage if not in cache
		if session == nil {
			internal.LogInfo("Session not in cache, reconstructing from storage...")
			// Load data using backend
			bubbles, err := backend.LoadBubbles()
			if err != nil {
				return fmt.Errorf("failed to load bubbles: %w", err)
			}

			composers, err := backend.LoadComposers()
			if err != nil {
				return fmt.Errorf("failed to load composers: %w", err)
			}

			contexts, err := backend.LoadMessageContexts()
			if err != nil {
				return fmt.Errorf("failed to load contexts: %w", err)
			}

			// Find the composer
			var targetComposer *internal.RawComposer
			for _, composer := range composers {
				if composer.ComposerID == sessionID {
					targetComposer = composer
					break
				}
			}

			if targetComposer == nil {
				return fmt.Errorf("session not found: %s", sessionID)
			}

			// Reconstruct conversation
			bubbleMap := internal.NewBubbleMap()
			for _, bubble := range bubbles {
				bubbleMap.Set(bubble.BubbleID, bubble)
			}

			reconstructor := internal.NewReconstructor(bubbleMap, contexts)
			conv, err := reconstructor.ReconstructConversation(targetComposer)
			if err != nil {
				return fmt.Errorf("failed to reconstruct conversation: %w", err)
			}

			// Associate with workspace
			workspaces, _ := internal.DetectWorkspaces(paths.BasePath)
			var composerContexts []*internal.MessageContext
			if ctxs, ok := contexts[conv.ComposerID]; ok {
				composerContexts = ctxs
			}
			assignedWorkspace := internal.AssociateComposerWithWorkspace(conv.ComposerID, composerContexts, workspaces)

			// Normalize
			normalizer := internal.NewNormalizer()
			session, err = normalizer.NormalizeConversation(conv, assignedWorkspace)
			if err != nil {
				return fmt.Errorf("failed to normalize conversation: %w", err)
			}

			// Save to cache for future use
			if err := cacheManager.SaveSessionAndUpdateIndex(session, cacheKey); err != nil {
				internal.LogWarn("Failed to save session to cache: %v", err)
			} else {
				internal.LogInfo("Session cached for faster future access")
			}
		}

		// Display session header
		displaySessionHeader(session)

		// Filter messages if needed
		messagesToShow := session.Messages
		var sinceTime *time.Time

		// Filter by timestamp if --since is provided
		if since != "" {
			parsedTime, err := time.Parse(time.RFC3339, since)
			if err != nil {
				return fmt.Errorf("invalid --since timestamp format (expected RFC3339): %w", err)
			}
			sinceTime = &parsedTime
			filtered := make([]internal.Message, 0, len(messagesToShow))
			for _, msg := range messagesToShow {
				if msg.Timestamp != "" {
					if msgTime, err := time.Parse(time.RFC3339, msg.Timestamp); err == nil {
						if msgTime.After(*sinceTime) || msgTime.Equal(*sinceTime) {
							filtered = append(filtered, msg)
						}
					}
				}
			}
			messagesToShow = filtered
		}

		// Apply limit if specified
		totalFiltered := len(messagesToShow)
		if limit > 0 && limit < len(messagesToShow) {
			messagesToShow = messagesToShow[:limit]
		}

		// Display messages
		for i, msg := range messagesToShow {
			displayMessage(i+1, msg, totalFiltered)
		}

		// Show remaining count if limit was applied
		if limit > 0 && limit < totalFiltered {
			remaining := totalFiltered - limit
			fmt.Println()
			fmt.Println(lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				Italic(true).
				Render(fmt.Sprintf("... (%d more message(s))", remaining)))
		}

		return nil
	},
}

func displaySessionHeader(session *internal.Session) {
	if session == nil {
		return
	}
	header := sessionHeaderStyle.Render(fmt.Sprintf("💬 %s", session.Metadata.Name))
	fmt.Println(header)

	// Create metadata line
	var metaParts []string
	if session.Metadata.CreatedAt != "" {
		metaParts = append(metaParts, fmt.Sprintf("Created: %s", session.Metadata.CreatedAt))
	}
	metaParts = append(metaParts, fmt.Sprintf("Messages: %d", len(session.Messages)))
	if session.Workspace != "" {
		metaParts = append(metaParts, fmt.Sprintf("Workspace: %s", session.Workspace))
	}

	if len(metaParts) > 0 {
		meta := sessionMetaStyle.Render(strings.Join(metaParts, " • "))
		fmt.Println(meta)
	}

	fmt.Println()
}

func displayMessage(index int, msg internal.Message, total int) {
	var actorStyle lipgloss.Style
	var actorLabel string

	switch msg.Actor {
	case "user":
		actorStyle = userMessageStyle
		actorLabel = "👤 User"
	case "assistant":
		actorStyle = assistantMessageStyle
		actorLabel = "🤖 Assistant"
	default:
		actorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
		actorLabel = fmt.Sprintf("🔧 %s", msg.Actor)
	}

	// Message header
	header := actorStyle.Render(actorLabel) + " " + timestampStyle.Render(fmt.Sprintf("[%d/%d]", index, total))
	if msg.Timestamp != "" {
		// Parse and format timestamp
		if t, err := time.Parse(time.RFC3339, msg.Timestamp); err == nil {
			header += " " + timestampStyle.Render(t.Format("15:04:05"))
		} else {
			header += " " + timestampStyle.Render(msg.Timestamp)
		}
	}

	fmt.Println(header)

	// Message content
	content := strings.TrimSpace(msg.Content)
	if content != "" {
		// Wrap long lines
		content = wrapText(content, 80)
		fmt.Println(messageContentStyle.Render(content))
	} else {
		fmt.Println(messageContentStyle.Foreground(lipgloss.Color("240")).Render("(empty message)"))
	}

	fmt.Println()
}

func wrapText(text string, width int) string {
	lines := strings.Split(text, "\n")
	var wrapped []string

	for _, line := range lines {
		if len(line) <= width {
			wrapped = append(wrapped, line)
			continue
		}

		// Wrap long lines
		words := strings.Fields(line)
		currentLine := ""
		for _, word := range words {
			if len(currentLine)+len(word)+1 > width {
				if currentLine != "" {
					wrapped = append(wrapped, currentLine)
					currentLine = word
				} else {
					wrapped = append(wrapped, word)
					currentLine = ""
				}
			} else {
				if currentLine == "" {
					currentLine = word
				} else {
					currentLine += " " + word
				}
			}
		}
		if currentLine != "" {
			wrapped = append(wrapped, currentLine)
		}
	}

	return strings.Join(wrapped, "\n")
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().IntVarP(&limit, "limit", "n", 0, "Limit number of messages to show")
	showCmd.Flags().StringVar(&since, "since", "", "Show messages since timestamp (ISO8601)")
}
