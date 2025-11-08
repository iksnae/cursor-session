package cmd

import (
	"fmt"
	"os"

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

		// Step 1: Detect storage paths
		fmt.Println(infoStyle.Render("Step 1: Detecting storage paths..."))
		paths, err := internal.DetectStoragePaths()
		if err != nil {
			fmt.Println(errorStyle.Render("❌ Failed to detect storage paths:"), err)
			os.Exit(1)
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
			}
			return fmt.Errorf("health check failed: no storage available")
		}
	},
}

func init() {
	rootCmd.AddCommand(healthcheckCmd)
	healthcheckCmd.Flags().BoolVarP(&healthcheckVerbose, "verbose", "v", false, "Show detailed diagnostic information")
}

