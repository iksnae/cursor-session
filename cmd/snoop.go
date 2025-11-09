package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
			agentPath, err := triggerCursorAgentHello()
			if err != nil {
				// Show where cursor-agent was found (if found) even on error
				if agentPath != "" {
					fmt.Printf("%s ℹ️  Found cursor-agent at: %s\n", snoopInfoStyle.Render(""), snoopPathStyle.Render(agentPath))
				}
				fmt.Printf("%s ⚠️  Could not invoke cursor-agent: %v\n", snoopWarningStyle.Render(""), err)
				fmt.Println(snoopInfoStyle.Render("   Continuing with path detection anyway..."))
			} else {
				if agentPath != "" {
					fmt.Printf("%s ✅ Found cursor-agent at: %s\n", snoopSuccessStyle.Render(""), snoopPathStyle.Render(agentPath))
				}
				fmt.Println(snoopSuccessStyle.Render("✅ Successfully invoked cursor-agent"))
				// Give it time to create the database - cursor-agent may need a moment
				fmt.Println(snoopInfoStyle.Render("   Waiting for database to be created..."))
				time.Sleep(5 * time.Second)
				
				// Re-check paths after waiting to see if database was created
				fmt.Println(snoopInfoStyle.Render("   Re-checking paths after database creation..."))
				
				// Force a fresh path detection after cursor-agent runs
				// This ensures we pick up any newly created directories
				time.Sleep(2 * time.Second)
			}
			fmt.Println()
		}

		// Get storage paths (with optional custom storage location)
		fmt.Println(snoopSectionStyle.Render("📂 Storage Path Detection"))
		paths, err := internal.GetStoragePaths(storagePath)
		if err != nil {
			fmt.Printf("%s ❌ Failed to get storage paths: %v\n", snoopErrorStyle.Render(""), err)
		} else {
			// Copy database files to temp location if --copy flag is set
			var cleanup func() error
			if copyDB {
				var copyErr error
				paths, cleanup, copyErr = internal.CopyStoragePaths(paths)
				if copyErr != nil {
					fmt.Printf("%s ❌ Failed to copy database files: %v\n", snoopErrorStyle.Render(""), copyErr)
				} else {
					fmt.Printf("%s ✅ Database files copied to temporary location\n", snoopSuccessStyle.Render(""))
					// Schedule cleanup when command completes
					defer func() {
						if cleanup != nil {
							if err := cleanup(); err != nil {
								fmt.Printf("⚠️  Failed to cleanup temporary files: %v\n", err)
							}
						}
					}()
				}
			}
			displayPathInfo(paths)
			
			// If --hello was used and we still don't see agent storage, check if directory was just created
			if snoopHello && !paths.HasAgentStorage() && paths.AgentStoragePath != "" {
				// Give it one more moment and check again
				time.Sleep(1 * time.Second)
				if info, err := os.Stat(paths.AgentStoragePath); err == nil && info.IsDir() {
					fmt.Printf("%s ✅ Agent storage directory now exists (created by cursor-agent)\n", snoopSuccessStyle.Render("  "))
					// Re-scan for databases
					if storeDBs, err := paths.FindAgentStoreDBs(); err == nil && len(storeDBs) > 0 {
						fmt.Printf("%s ✅ Found %d store.db file(s) after cursor-agent run\n", snoopSuccessStyle.Render("  "), len(storeDBs))
					}
				}
			}
		}
		fmt.Println()

		// Try alternative paths
		fmt.Println(snoopSectionStyle.Render("🔎 Alternative Path Search"))
		checkAlternativePaths()
		fmt.Println()

		// Deep search for database files
		fmt.Println(snoopSectionStyle.Render("🔍 Deep Search for Database Files"))
		deepSearchForDatabases()
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
	home, _ := os.UserHomeDir()
	agentStoragePaths := []string{
		filepath.Join(home, ".config/cursor/chats"), // Newer location (CI/GH workflows)
		filepath.Join(home, ".cursor/chats"),        // Older location (local installs)
	}
	
	foundAgentStorage := false
	for _, agentPath := range agentStoragePaths {
		fmt.Printf("  %s\n", snoopPathStyle.Render(agentPath))
		if info, err := os.Stat(agentPath); err == nil && info.IsDir() {
			foundAgentStorage = true
			fmt.Printf("  %s\n", snoopSuccessStyle.Render("✅ Directory exists"))
			// Create a temporary StoragePaths to use FindAgentStoreDBs
			tempPaths := internal.StoragePaths{AgentStoragePath: agentPath}
			storeDBs, err := tempPaths.FindAgentStoreDBs()
			if err != nil {
				fmt.Printf("  %s ❌ Error scanning: %v\n", snoopErrorStyle.Render(""), err)
			} else if len(storeDBs) > 0 {
				fmt.Printf("  %s ✅ Found %d store.db file(s)\n", snoopSuccessStyle.Render(""), len(storeDBs))
				for i, db := range storeDBs {
					if i < 3 {
						fmt.Printf("    • %s\n", snoopPathStyle.Render(db))
					}
				}
				if len(storeDBs) > 3 {
					fmt.Printf("    ... and %d more\n", len(storeDBs)-3)
				}
			} else {
				fmt.Printf("  %s ⚠️  Directory exists but no store.db files found\n", snoopWarningStyle.Render(""))
			}
			break // Found the active location, no need to check others
		} else {
			fmt.Printf("  %s\n", snoopWarningStyle.Render("⚠️  Does not exist"))
		}
	}
	
	if !foundAgentStorage && runtime.GOOS == "linux" {
		fmt.Printf("  %s\n", snoopWarningStyle.Render("⚠️  No agent storage directories found"))
	} else if runtime.GOOS != "linux" {
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

func deepSearchForDatabases() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(snoopWarningStyle.Render("⚠️  Could not get home directory"))
		return
	}

	fmt.Println(snoopInfoStyle.Render("Searching for database files in likely locations..."))

	var foundDBs []struct {
		path string
		typ  string
	}

	// First, specifically check cursor-agent storage directories (check both locations)
	// Priority: .config/cursor/chats (newer location used in CI/GH workflows) then .cursor/chats
	cursorChatsDirs := []string{
		filepath.Join(home, ".config", "cursor", "chats"), // Newer location (CI/GH workflows)
		filepath.Join(home, ".cursor", "chats"),            // Older location (local installs)
	}
	
	for _, cursorChatsDir := range cursorChatsDirs {
		if info, err := os.Stat(cursorChatsDir); err == nil && info.IsDir() {
			// Walk the chats directory looking for store.db files
			err := filepath.Walk(cursorChatsDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if !info.IsDir() && info.Name() == "store.db" {
					foundDBs = append(foundDBs, struct {
						path string
						typ  string
					}{path: path, typ: "store.db (cursor-agent)"})
				}
				return nil
			})
			if err == nil && len(foundDBs) > 0 {
				// Found databases, no need to search further
				fmt.Printf("%s ✅ Found %d database file(s) in %s:\n", snoopSuccessStyle.Render("  "), len(foundDBs), cursorChatsDir)
				for i, db := range foundDBs {
					if i < 10 {
						fmt.Printf("    • %s\n", snoopPathStyle.Render(db.path))
					}
				}
				if len(foundDBs) > 10 {
					fmt.Printf("    ... and %d more\n", len(foundDBs)-10)
				}
				return
			}
		}
	}

	// Target specific directories where Cursor databases are likely to be
	searchDirs := []string{
		filepath.Join(home, ".config"),
		filepath.Join(home, ".local"),
		filepath.Join(home, ".cursor"),
		filepath.Join(home, "Library", "Application Support"), // macOS
	}

	// Also check XDG directories if set
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		searchDirs = append(searchDirs, xdgConfig)
	}
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		searchDirs = append(searchDirs, xdgData)
	}

	for _, searchDir := range searchDirs {
		if _, err := os.Stat(searchDir); err != nil {
			continue // Skip if directory doesn't exist
		}

		err := filepath.Walk(searchDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}

			// Skip common directories that won't have databases
			if info.IsDir() {
				base := filepath.Base(path)
				if base == "node_modules" || base == ".git" || base == ".cache" || base == ".npm" || base == "Cache" {
					return filepath.SkipDir
				}
				return nil
			}

			// Look for database files
			if info.Name() == "state.vscdb" || info.Name() == "store.db" {
				typ := "state.vscdb"
				if info.Name() == "store.db" {
					typ = "store.db"
				}
				foundDBs = append(foundDBs, struct {
					path string
					typ  string
				}{path: path, typ: typ})
			}

			return nil
		})

		if err != nil {
			// Silently skip errors - some directories might not be accessible
			continue
		}
	}

	if len(foundDBs) > 0 {
		fmt.Printf("%s ✅ Found %d database file(s):\n", snoopSuccessStyle.Render("  "), len(foundDBs))
		for i, db := range foundDBs {
			if i < 10 { // Show first 10
				fmt.Printf("    • %s (%s)\n", snoopPathStyle.Render(db.path), db.typ)
			}
		}
		if len(foundDBs) > 10 {
			fmt.Printf("    ... and %d more\n", len(foundDBs)-10)
		}
	} else {
		fmt.Printf("  %s\n", snoopWarningStyle.Render("⚠️  No database files found in likely locations"))
		fmt.Printf("  %s\n", snoopInfoStyle.Render("  Searched: .config, .local, .cursor, Library/Application Support, XDG directories"))
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
		fmt.Println(snoopInfoStyle.Render("💡 Tips:"))
		fmt.Println(snoopInfoStyle.Render("  • Use --hello flag to seed the database with cursor-agent"))
		fmt.Println(snoopInfoStyle.Render("  • Make sure cursor-agent is authenticated: run 'cursor-agent login'"))
		fmt.Println(snoopInfoStyle.Render("  • In CI environments, Cursor databases won't be found (this is expected)"))
	}
}

// triggerCursorAgentHello invokes cursor-agent with a simple "hello" prompt to seed the database
// Returns the path where cursor-agent was found, or an error
func triggerCursorAgentHello() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Find cursor-agent in common locations (check installed locations first, then PATH)
	possiblePaths := []string{
		filepath.Join(home, ".local/bin/cursor-agent"),  // Most common Linux location
		filepath.Join(home, ".cursor/bin/cursor-agent"), // Alternative location
		"cursor-agent", // In PATH (check last)
	}

	// On macOS, also check common installation locations
	if runtime.GOOS == "darwin" {
		possiblePaths = append([]string{
			"/usr/local/bin/cursor-agent",
			"/opt/homebrew/bin/cursor-agent",
		}, possiblePaths...)
	}

	var cursorAgentPath string
	var foundLocation string
	for _, path := range possiblePaths {
		if path == "cursor-agent" {
			// Check if it's in PATH
			if fullPath, err := exec.LookPath("cursor-agent"); err == nil {
				cursorAgentPath = "cursor-agent"
				foundLocation = fullPath
				break
			}
		} else {
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				cursorAgentPath = path
				foundLocation = path
				break
			}
		}
	}

	if cursorAgentPath == "" {
		return "", fmt.Errorf("cursor-agent not found in PATH or common locations")
	}

	// Check if CURSOR_API_KEY is set (for non-interactive authentication)
	hasAPIKey := os.Getenv("CURSOR_API_KEY") != ""
	
	// Only check authentication status if no API key is set
	// (API key authentication doesn't require interactive login)
	if !hasAPIKey {
		checkCmd := exec.Command(cursorAgentPath, "status")
		checkCmd.Env = os.Environ()
		var checkStderr bytes.Buffer
		checkCmd.Stderr = &checkStderr
		checkCmd.Stdout = &checkStderr
		if err := checkCmd.Run(); err != nil {
			// If status check fails, it might mean not authenticated
			stderrStr := checkStderr.String()
			if strings.Contains(stderrStr, "Authentication required") || 
			   strings.Contains(stderrStr, "login") ||
			   strings.Contains(stderrStr, "not authenticated") {
				return foundLocation, fmt.Errorf("cursor-agent found at %s but requires authentication (run 'cursor-agent login' or set CURSOR_API_KEY environment variable)", foundLocation)
			}
			// Other errors - continue anyway, might still work
		}
	}

	// Run cursor-agent with a simple prompt to trigger session creation
	// Use a context with timeout to avoid hanging
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cursorAgentPath, "-p", "hello", "--model", "auto", "--print")
	cmd.Env = os.Environ()
	
	// Capture stderr to detect authentication errors
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = os.Stderr // Redirect stdout to stderr to avoid cluttering

	// Start the command
	if err := cmd.Start(); err != nil {
		return foundLocation, fmt.Errorf("failed to start cursor-agent: %w", err)
	}

	// Wait for completion with timeout - this ensures the process finishes
	// and gives it time to create the database
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
		case err := <-done:
			if err != nil {
				// Check if it's an authentication error
				stderrStr := stderr.String()
				if strings.Contains(stderrStr, "Authentication required") || 
				   strings.Contains(stderrStr, "login") ||
				   strings.Contains(stderrStr, "CURSOR_API_KEY") {
					return foundLocation, fmt.Errorf("cursor-agent requires authentication: %w (run 'cursor-agent login' or set CURSOR_API_KEY)", err)
				}
				// Other errors might still allow database creation, so don't fail completely
				return foundLocation, nil
			}
		case <-ctx.Done():
			// Timeout reached, but that's okay - the process might still be creating the database
			_ = cmd.Process.Kill()
	}

	return foundLocation, nil
}

func init() {
	rootCmd.AddCommand(snoopCmd)
	snoopCmd.Flags().BoolVar(&snoopHello, "hello", false, "Invoke cursor-agent with a simple prompt to seed the database")
}

