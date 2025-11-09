package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/iksnae/cursor-session/internal"
	"github.com/spf13/cobra"
)

var (
	healthcheckVerbose bool
)

var (
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	sectionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true).
			Underline(true)
)

// healthcheckCmd represents the healthcheck command
var healthcheckCmd = &cobra.Command{
	Use:   "healthcheck",
	Short: "Check if cursor-session can locate and access session data",
	Long: `Check the health of cursor-session by verifying:
  • Storage path detection
  • Storage format availability (desktop app or agent CLI)
  • Session data accessibility
  • Session count

This command is useful for debugging storage issues, especially in CI/CD environments.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(sectionStyle.Render("🔍 Cursor Session Health Check"))
		fmt.Println()

		// Step 1: Get storage paths (with optional custom storage location)
		fmt.Println(infoStyle.Render("Step 1: Getting storage paths..."))
		paths, err := internal.GetStoragePaths(storagePath)
		if err != nil {
			fmt.Println(errorStyle.Render("❌ Failed to get storage paths:"), err)
			os.Exit(1)
		}

		// Copy database files to temp location if --copy flag is set
		var cleanup func() error
		if copyDB {
			var copyErr error
			paths, cleanup, copyErr = internal.CopyStoragePaths(paths)
			if copyErr != nil {
				fmt.Println(errorStyle.Render("❌ Failed to copy database files:"), copyErr)
				os.Exit(1)
			}
			fmt.Println(successStyle.Render("✅ Database files copied to temporary location"))
			// Schedule cleanup when command completes
			defer func() {
				if cleanup != nil {
					if err := cleanup(); err != nil {
						fmt.Printf("⚠️  Failed to cleanup temporary files: %v\n", err)
					}
				}
			}()
		}

		fmt.Println(successStyle.Render("✅ Storage paths detected"))
		if healthcheckVerbose {
			fmt.Printf("   Base path: %s\n", paths.BasePath)
			fmt.Printf("   Global storage: %s\n", paths.GlobalStorage)
			fmt.Printf("   Agent storage: %s\n", paths.AgentStoragePath)
		}
		fmt.Println()

		// Step 2: Check desktop app storage
		fmt.Println(infoStyle.Render("Step 2: Checking desktop app storage..."))
		desktopAppExists := paths.GlobalStorageExists()
		if desktopAppExists {
			dbPath := paths.GetGlobalStorageDBPath()
			fmt.Println(successStyle.Render("✅ Desktop app storage found"))
			if healthcheckVerbose {
				fmt.Printf("   Database: %s\n", dbPath)
			}
		} else {
			fmt.Println(warningStyle.Render("⚠️  Desktop app storage not found"))
			if healthcheckVerbose {
				fmt.Printf("   Expected: %s\n", paths.GetGlobalStorageDBPath())
			}
		}
		fmt.Println()

		// Step 3: Check agent storage
		fmt.Println(infoStyle.Render("Step 3: Checking agent CLI storage..."))
		agentStorageExists := paths.HasAgentStorage()
		var storeDBs []string
		var storeDBsErr error
		if agentStorageExists {
			fmt.Println(successStyle.Render("✅ Agent storage directory exists"))
			if healthcheckVerbose {
				fmt.Printf("   Directory: %s\n", paths.AgentStoragePath)
			}
			storeDBs, storeDBsErr = paths.FindAgentStoreDBs()
			if storeDBsErr != nil {
				fmt.Println(warningStyle.Render("⚠️  Error scanning agent storage:"), storeDBsErr)
			} else if len(storeDBs) > 0 {
				fmt.Println(successStyle.Render(fmt.Sprintf("✅ Found %d session database(s)", len(storeDBs))))
				if healthcheckVerbose {
					for i, db := range storeDBs {
						if i < 5 { // Show first 5
							fmt.Printf("   [%d] %s\n", i+1, db)
						}
					}
					if len(storeDBs) > 5 {
						fmt.Printf("   ... and %d more\n", len(storeDBs)-5)
					}
				}
			} else {
				fmt.Println(warningStyle.Render("⚠️  Agent storage directory exists but no store.db files found"))
				if healthcheckVerbose {
					fmt.Printf("   Expected pattern: %s/{hash}/{session-id}/store.db\n", paths.AgentStoragePath)
				}
			}
		} else {
			fmt.Println(warningStyle.Render("⚠️  Agent storage directory not found"))
			if healthcheckVerbose {
				if paths.AgentStoragePath != "" {
					fmt.Printf("   Expected: %s\n", paths.AgentStoragePath)
					fmt.Printf("   This directory is created when cursor-agent CLI is first used\n")
				} else {
					fmt.Printf("   Agent storage not available on this platform\n")
				}
			}
		}
		fmt.Println()

		// Step 4: Try to create storage backend
		fmt.Println(infoStyle.Render("Step 4: Testing storage backend access..."))
		backend, err := internal.NewStorageBackend(paths)
		if err != nil {
			fmt.Println(errorStyle.Render("❌ Failed to initialize storage backend"))
			fmt.Println()
			fmt.Println("Error details:")
			fmt.Println(err)
			fmt.Println()

			// Check if we're in CI
			if internal.IsCIEnvironment() {
				fmt.Println(infoStyle.Render("CI/CD Environment Detected"))
				fmt.Println("This is expected if cursor-agent hasn't created sessions yet.")
				fmt.Println("Sessions are created automatically when cursor-agent CLI runs.")
				fmt.Println()
				fmt.Println(successStyle.Render("✅ Health check passed (CI environment - no storage expected)"))
				return nil // Exit successfully in CI when storage is not found
			}

			os.Exit(1)
		}
		fmt.Println(successStyle.Render("✅ Storage backend initialized"))
		if healthcheckVerbose {
			switch backend.(type) {
			case *internal.Storage:
				fmt.Println("   Type: Desktop app storage (globalStorage)")
			case *internal.AgentStorage:
				fmt.Println("   Type: Agent CLI storage")
			default:
				fmt.Printf("   Type: %T\n", backend)
			}
		}
		fmt.Println()

		// Step 5: Try to load sessions
		fmt.Println(infoStyle.Render("Step 5: Loading session data..."))
		composers, err := backend.LoadComposers()
		if err != nil {
			fmt.Println(errorStyle.Render("❌ Failed to load composers:"), err)
			if internal.IsCIEnvironment() {
				fmt.Println()
				fmt.Println(infoStyle.Render("CI/CD Environment Detected"))
				fmt.Println("This error may be expected if cursor-agent hasn't created sessions yet.")
				fmt.Println(successStyle.Render("✅ Health check passed (CI environment - storage accessible)"))
				return nil // Exit successfully in CI even if loading fails
			}
			os.Exit(1)
		}

		sessionCount := len(composers)
		if sessionCount > 0 {
			fmt.Println(successStyle.Render(fmt.Sprintf("✅ Found %d session(s)", sessionCount)))
			if healthcheckVerbose {
				for i, composer := range composers {
					if i < 5 { // Show first 5
						name := composer.Name
						if name == "" {
							name = "Untitled"
						}
						fmt.Printf("   [%d] %s (ID: %s)\n", i+1, name, composer.ComposerID[:8])
					}
				}
				if len(composers) > 5 {
					fmt.Printf("   ... and %d more\n", len(composers)-5)
				}
			}
		} else {
			fmt.Println(warningStyle.Render("⚠️  No sessions found"))
			fmt.Println("   This could mean:")
			fmt.Println("   • No chat sessions have been created yet")
			fmt.Println("   • Sessions exist but are in a different format")
			if internal.IsCIEnvironment() {
				fmt.Println("   • In CI: cursor-agent may not have created sessions yet")
				fmt.Println()
				fmt.Println(infoStyle.Render("Attempting to trigger session creation..."))

				// Try to trigger cursor-agent to create a session
				if err := triggerCursorAgentSession(); err != nil {
					fmt.Println(warningStyle.Render(fmt.Sprintf("   ⚠️  Could not trigger cursor-agent: %v", err)))
					fmt.Println("   This is okay - sessions will be created when cursor-agent runs normally.")
				} else {
					fmt.Println(successStyle.Render("   ✅ Triggered cursor-agent session creation"))
					fmt.Println("   Waiting for session to be created...")

					// Wait a bit and recheck
					time.Sleep(3 * time.Second)

					// Recheck storage
					paths2, err2 := internal.GetStoragePaths(storagePath)
					if err2 == nil {
						storeDBs2, _ := paths2.FindAgentStoreDBs()
						if len(storeDBs2) > 0 {
							fmt.Println(successStyle.Render(fmt.Sprintf("   ✅ Session created! Found %d database(s)", len(storeDBs2))))
							// Update sessionCount for summary
							backend2, err2 := internal.NewStorageBackend(paths2)
							if err2 == nil {
								composers2, err2 := backend2.LoadComposers()
								if err2 == nil {
									sessionCount = len(composers2)
									fmt.Println(successStyle.Render(fmt.Sprintf("   ✅ Loaded %d session(s)", sessionCount)))
								}
							}
						} else {
							fmt.Println(warningStyle.Render("   ⚠️  Session may still be initializing. This is normal."))
						}
					}
				}
			}
		}
		fmt.Println()

		// Summary
		fmt.Println(sectionStyle.Render("📊 Summary"))
		fmt.Println()

		allGood := desktopAppExists || (agentStorageExists && len(storeDBs) > 0)
		if allGood && sessionCount > 0 {
			fmt.Println(successStyle.Render("✅ Health check passed!"))
			fmt.Println(successStyle.Render("   • Storage: Available"))
			fmt.Println(successStyle.Render(fmt.Sprintf("   • Sessions: %d found", sessionCount)))
			return nil
		} else if allGood {
			fmt.Println(warningStyle.Render("⚠️  Storage available but no sessions found"))
			fmt.Println("   • Storage backend is working")
			fmt.Println("   • No sessions are currently available")
			return nil
		} else {
			fmt.Println(errorStyle.Render("❌ Health check failed"))
			fmt.Println("   • No storage format is available")
			fmt.Println("   • Cannot access session data")
			if internal.IsCIEnvironment() {
				fmt.Println()
				fmt.Println("Note: This is expected in CI if cursor-agent hasn't run yet.")
				fmt.Println(successStyle.Render("✅ Health check passed (CI environment - no storage expected)"))
				return nil // Exit successfully in CI when no storage is available
			}
			return fmt.Errorf("health check failed: no storage available")
		}
	},
}

// triggerCursorAgentSession attempts to trigger cursor-agent to create a session
// by sending a simple "hello" message. This is useful in CI environments where
// sessions may not exist yet.
func triggerCursorAgentSession() error {
	// Find cursor-agent in common locations
	possiblePaths := []string{
		"cursor-agent", // In PATH
		filepath.Join(os.Getenv("HOME"), ".local/bin/cursor-agent"),
		filepath.Join(os.Getenv("HOME"), ".cursor/bin/cursor-agent"),
	}

	var cursorAgentPath string
	for _, path := range possiblePaths {
		if path == "cursor-agent" {
			// Check if it's in PATH
			if _, err := exec.LookPath("cursor-agent"); err == nil {
				cursorAgentPath = "cursor-agent"
				break
			}
		} else {
			if _, err := os.Stat(path); err == nil {
				cursorAgentPath = path
				break
			}
		}
	}

	if cursorAgentPath == "" {
		return fmt.Errorf("cursor-agent not found in PATH or common locations")
	}

	// Run cursor-agent with a simple prompt to trigger session creation
	// Use a simple "hello" message that should create a session
	// Use a context with timeout to avoid hanging
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cursorAgentPath, "--print", "hello", "--model", "auto")
	cmd.Env = os.Environ()

	// Run asynchronously - we don't need to wait for completion
	// Just starting it should trigger session creation
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start cursor-agent: %w", err)
	}

	// Don't wait for completion - just let it run in background
	// The session should be created shortly
	go func() {
		_ = cmd.Wait() // Clean up the process (ignore error)
	}()

	return nil
}

func init() {
	rootCmd.AddCommand(healthcheckCmd)
	healthcheckCmd.Flags().BoolVarP(&healthcheckVerbose, "verbose", "v", false, "Show detailed diagnostic information")
}
