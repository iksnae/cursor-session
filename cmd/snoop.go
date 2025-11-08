package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/iksnae/cursor-session/internal"
	"github.com/spf13/cobra"
)

var (
	snoopHello bool
)

var (
	snoopSuccessStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)

	snoopWarningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	snoopErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	snoopInfoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("39"))

	snoopSectionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("62")).
		Bold(true).
		Underline(true)

	snoopPathStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("243"))
)

// snoopCmd represents the snoop command
var snoopCmd = &cobra.Command{
	Use:   "snoop",
	Short: "Attempt to find the correct path to cursor database files",
	Long: `Snoop attempts to locate Cursor database files across different operating systems.

This command will:
  • Check standard storage paths for your OS
  • Verify if database files exist at those locations
  • Display detailed information about what was found
  • Optionally seed the database with --hello flag

The --hello flag will invoke cursor-agent with a simple prompt to create a session,
which can help seed the database if it doesn't exist yet.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If --hello flag is set, trigger cursor-agent first
		if snoopHello {
			fmt.Println(snoopInfoStyle.Render("🔍 Invoking cursor-agent to seed database..."))
			if err := triggerCursorAgentHello(); err != nil {
				fmt.Printf("%s ⚠️  Could not invoke cursor-agent: %v\n", snoopWarningStyle.Render(""), err)
				fmt.Println(snoopInfoStyle.Render("   Continuing with path detection anyway..."))
			} else {
				fmt.Println(snoopSuccessStyle.Render("✅ Successfully invoked cursor-agent"))
				// Give it a moment to create the database
				fmt.Println(snoopInfoStyle.Render("   Waiting a moment for database to be created..."))
				time.Sleep(2 * time.Second)
			}
			fmt.Println()
		}

		// Detect standard paths
		fmt.Println(snoopSectionStyle.Render("📂 Standard Path Detection"))
		paths, err := internal.DetectStoragePaths()
		if err != nil {
			fmt.Printf("%s ❌ Failed to detect storage paths: %v\n", snoopErrorStyle.Render(""), err)
		} else {
			displayPathInfo(paths)
		}
		fmt.Println()

		// Try alternative paths
		fmt.Println(snoopSectionStyle.Render("🔎 Alternative Path Search"))
		checkAlternativePaths()
		fmt.Println()

		// Summary
		fmt.Println(snoopSectionStyle.Render("📊 Summary"))
		displaySummary(paths)

		return nil
	},
}

func displayPathInfo(paths internal.StoragePaths) {
	fmt.Println(snoopInfoStyle.Render("Base Path:"))
	fmt.Printf("  %s\n", snoopPathStyle.Render(paths.BasePath))
	checkPath(paths.BasePath, "  ")

	fmt.Println()
	fmt.Println(snoopInfoStyle.Render("Global Storage:"))
	fmt.Printf("  %s\n", snoopPathStyle.Render(paths.GlobalStorage))
	checkPath(paths.GlobalStorage, "  ")

		// Check for state.vscdb in globalStorage
		dbPath := paths.GetGlobalStorageDBPath()
		fmt.Printf("  Database: %s\n", snoopPathStyle.Render(dbPath))
		if paths.GlobalStorageExists() {
			fmt.Printf("  %s\n", snoopSuccessStyle.Render("✅ Database file exists"))
			// Try to open it
			if db, err := internal.OpenDatabase(dbPath); err == nil {
				db.Close()
				fmt.Printf("  %s\n", snoopSuccessStyle.Render("✅ Database is accessible"))
			} else {
				fmt.Printf("%s ⚠️  Database exists but cannot be opened: %v\n", snoopWarningStyle.Render("  "), err)
			}
		} else {
			fmt.Printf("  %s\n", snoopWarningStyle.Render("⚠️  Database file does not exist"))
		}

	fmt.Println()
	fmt.Println(snoopInfoStyle.Render("Workspace Storage:"))
	fmt.Printf("  %s\n", snoopPathStyle.Render(paths.WorkspaceStorage))
	checkPath(paths.WorkspaceStorage, "  ")

	// Check for state.vscdb files in workspaceStorage subdirectories
	if info, err := os.Stat(paths.WorkspaceStorage); err == nil && info.IsDir() {
		var dbCount int
		err := filepath.Walk(paths.WorkspaceStorage, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && info.Name() == "state.vscdb" {
				dbCount++
			}
			return nil
		})
		if err != nil {
			fmt.Printf("%s ⚠️  Error scanning workspace storage: %v\n", snoopWarningStyle.Render("  "), err)
		} else if dbCount > 0 {
			fmt.Printf("%s ✅ Found %d state.vscdb file(s) in subdirectories\n", snoopSuccessStyle.Render("  "), dbCount)
		} else {
			fmt.Printf("  %s\n", snoopWarningStyle.Render("⚠️  No state.vscdb files found in subdirectories"))
		}
	}

	fmt.Println()
	fmt.Println(snoopInfoStyle.Render("Agent Storage:"))
	if paths.AgentStoragePath != "" {
		fmt.Printf("  %s\n", snoopPathStyle.Render(paths.AgentStoragePath))
		checkPath(paths.AgentStoragePath, "  ")

		if paths.HasAgentStorage() {
			storeDBs, err := paths.FindAgentStoreDBs()
			if err != nil {
				fmt.Printf("%s ⚠️  Error scanning: %v\n", snoopWarningStyle.Render("  "), err)
			} else if len(storeDBs) > 0 {
				fmt.Printf("%s ✅ Found %d store.db file(s)\n", snoopSuccessStyle.Render("  "), len(storeDBs))
				for i, db := range storeDBs {
					if i < 3 { // Show first 3
						fmt.Printf("    • %s\n", snoopPathStyle.Render(db))
					}
				}
				if len(storeDBs) > 3 {
					fmt.Printf("    ... and %d more\n", len(storeDBs)-3)
				}
			} else {
				fmt.Printf("  %s\n", snoopWarningStyle.Render("⚠️  Directory exists but no store.db files found"))
			}
		} else {
			fmt.Printf("  %s\n", snoopWarningStyle.Render("⚠️  Directory does not exist"))
		}
	} else {
		fmt.Printf("  %s\n", snoopInfoStyle.Render("ℹ️  Not available on this OS (Linux only)"))
	}
}

func checkPath(path string, indent string) {
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			fmt.Printf("%s%s\n", indent, snoopSuccessStyle.Render("✅ Directory exists"))
		} else {
			fmt.Printf("%s%s\n", indent, snoopSuccessStyle.Render("✅ File exists"))
		}
	} else if os.IsNotExist(err) {
		fmt.Printf("%s%s\n", indent, snoopWarningStyle.Render("⚠️  Does not exist"))
	} else {
		fmt.Printf("%s%s ❌ Error checking: %v\n", indent, snoopErrorStyle.Render(""), err)
	}
}

func checkAlternativePaths() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(snoopWarningStyle.Render("⚠️  Could not get home directory"))
		return
	}

	// Try various alternative locations
	alternatives := []struct {
		name string
		path string
	}{
		{"Windows-style config (if on Linux)", filepath.Join(home, "AppData", "Roaming", "Cursor", "User")},
		{"Alternative Linux config", filepath.Join(home, ".cursor", "User")},
		{"Alternative macOS location", filepath.Join(home, "Library", "Preferences", "Cursor", "User")},
		{"XDG config home (if set)", filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "Cursor", "User")},
		{"XDG data home (if set)", filepath.Join(os.Getenv("XDG_DATA_HOME"), "Cursor", "User")},
	}

	foundAny := false
	for _, alt := range alternatives {
		if alt.path == "" {
			continue
		}
		fmt.Printf("%s: %s\n", snoopInfoStyle.Render(alt.name), snoopPathStyle.Render(alt.path))
		if _, err := os.Stat(alt.path); err == nil {
			fmt.Printf("  %s\n", snoopSuccessStyle.Render("✅ Found!"))
			foundAny = true

			// Check for database files
			globalStoragePath := filepath.Join(alt.path, "globalStorage")
			dbPath := filepath.Join(globalStoragePath, "state.vscdb")
			if _, err := os.Stat(dbPath); err == nil {
				fmt.Printf("%s ✅ Database found: %s\n", snoopSuccessStyle.Render("  "), dbPath)
			}
		} else {
			fmt.Printf("  %s\n", snoopWarningStyle.Render("⚠️  Not found"))
		}
	}

	if !foundAny {
		fmt.Println(snoopInfoStyle.Render("ℹ️  No alternative paths found"))
	}
}

func displaySummary(paths internal.StoragePaths) {
	var found []string
	var missing []string

	// Check globalStorage
	if paths.GlobalStorageExists() {
		found = append(found, "Desktop app storage (globalStorage)")
	} else {
		missing = append(missing, "Desktop app storage (globalStorage)")
	}

	// Check agent storage
	if paths.HasAgentStorage() {
		storeDBs, _ := paths.FindAgentStoreDBs()
		if len(storeDBs) > 0 {
			found = append(found, fmt.Sprintf("Agent storage (%d session(s))", len(storeDBs)))
		} else {
			missing = append(missing, "Agent storage (directory exists but no sessions)")
		}
	} else if paths.AgentStoragePath != "" {
		missing = append(missing, "Agent storage (directory does not exist)")
	}

	if len(found) > 0 {
		fmt.Println(snoopSuccessStyle.Render("✅ Found storage:"))
		for _, item := range found {
			fmt.Printf("  • %s\n", item)
		}
	}

	if len(missing) > 0 {
		fmt.Println()
		fmt.Println(snoopWarningStyle.Render("⚠️  Missing storage:"))
		for _, item := range missing {
			fmt.Printf("  • %s\n", item)
		}
	}

	if len(found) == 0 && len(missing) > 0 {
		fmt.Println()
		fmt.Println(snoopInfoStyle.Render("💡 Tip: Use --hello flag to seed the database with cursor-agent"))
	}
}

// triggerCursorAgentHello invokes cursor-agent with a simple "hello" prompt to seed the database
func triggerCursorAgentHello() error {
	// Find cursor-agent in common locations
	possiblePaths := []string{
		"cursor-agent", // In PATH
		filepath.Join(os.Getenv("HOME"), ".local/bin/cursor-agent"),
		filepath.Join(os.Getenv("HOME"), ".cursor/bin/cursor-agent"),
	}

	// On macOS, also check common installation locations
	if runtime.GOOS == "darwin" {
		possiblePaths = append(possiblePaths,
			"/usr/local/bin/cursor-agent",
			"/opt/homebrew/bin/cursor-agent",
		)
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
	// Use a context with timeout to avoid hanging
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cursorAgentPath, "-p", "hello", "--model", "auto", "--print")
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
	rootCmd.AddCommand(snoopCmd)
	snoopCmd.Flags().BoolVar(&snoopHello, "hello", false, "Invoke cursor-agent with a simple prompt to seed the database")
}

