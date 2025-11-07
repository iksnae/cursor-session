package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/iksnae/cursor-session/internal"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade cursor-session to the latest version",
	Long: `Upgrade cursor-session by pulling the latest changes from the repository
and reinstalling the binary.

This command will:
1. Find the repository (if installed from source)
2. Pull latest changes from git
3. Rebuild the binary
4. Reinstall to the current installation location`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the current binary path
		currentBinary, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get current binary path: %w", err)
		}

		// Resolve symlinks to get the real path
		realPath, err := filepath.EvalSymlinks(currentBinary)
		if err == nil {
			currentBinary = realPath
		}

		// Try to find the repository
		repoPath, err := findRepository()
		if err != nil {
			helpMsg := "If you installed via 'go install', you can upgrade by running:\n" +
				"  go install github.com/iksnae/cursor-session@main\n\n" +
				"Or if you cloned the repo, make sure you're in the repository directory"
			return fmt.Errorf("failed to find repository: %w\n\n%s", err, helpMsg)
		}

		// Check if git is available
		if _, err := exec.LookPath("git"); err != nil {
			return fmt.Errorf("git is not installed or not in PATH")
		}

		// Check if Go is available
		if _, err := exec.LookPath("go"); err != nil {
			return fmt.Errorf("go is not installed or not in PATH")
		}

		// Change to repository directory
		originalDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		defer os.Chdir(originalDir)

		if err := os.Chdir(repoPath); err != nil {
			return fmt.Errorf("failed to change to repository directory: %w", err)
		}

		// Check if we're in a git repository
		if _, err := exec.Command("git", "rev-parse", "--git-dir").Output(); err != nil {
			return fmt.Errorf("not a git repository: %s", repoPath)
		}

		ctx := context.Background()
		steps := []internal.ProgressStep{}

		// Check if there's a remote configured
		remotes, err := exec.Command("git", "remote").Output()
		if err == nil && len(remotes) > 0 {
			steps = append(steps, internal.ProgressStep{
				Message: "Pulling latest changes from repository",
				Fn: func() error {
					pullCmd := exec.Command("git", "pull")
					pullCmd.Stdout = os.Stdout
					pullCmd.Stderr = os.Stderr
					if err := pullCmd.Run(); err != nil {
						return fmt.Errorf("failed to pull latest changes: %w", err)
					}
					return nil
				},
			})
		}

		steps = append(steps,
			internal.ProgressStep{
				Message: "Building new binary",
				Fn: func() error {
					buildCmd := exec.Command("go", "build", "-buildvcs=false", "-o", "cursor-session", ".")
					buildCmd.Stdout = os.Stdout
					buildCmd.Stderr = os.Stderr
					if err := buildCmd.Run(); err != nil {
						return fmt.Errorf("failed to build binary: %w", err)
					}

					// Check if the build was successful
					if _, err := os.Stat("cursor-session"); err != nil {
						return fmt.Errorf("binary was not created after build")
					}
					return nil
				},
			},
			internal.ProgressStep{
				Message: fmt.Sprintf("Installing to %s", currentBinary),
				Fn: func() error {
					// Make sure the target directory exists
					targetDir := filepath.Dir(currentBinary)
					if err := os.MkdirAll(targetDir, 0755); err != nil {
						return fmt.Errorf("failed to create target directory: %w", err)
					}

					// Copy the new binary
					newBinaryPath := filepath.Join(repoPath, "cursor-session")
					if err := copyFile(newBinaryPath, currentBinary); err != nil {
						return fmt.Errorf("failed to install binary: %w", err)
					}

					// Make it executable
					if err := os.Chmod(currentBinary, 0755); err != nil {
						return fmt.Errorf("failed to make binary executable: %w", err)
					}
					return nil
				},
			},
			internal.ProgressStep{
				Message: "Verifying installation",
				Fn: func() error {
					verifyCmd := exec.Command(currentBinary, "--version")
					output, err := verifyCmd.Output()
					if err != nil {
						internal.LogWarn("Installation completed but verification failed: %v", err)
						return nil // Don't fail on verification
					}
					fmt.Println()
					fmt.Println("New version:")
					fmt.Print(string(output))
					return nil
				},
			},
		)

		if err := internal.ShowProgressWithSteps(ctx, steps); err != nil {
			return err
		}

		internal.PrintSuccess("Upgrade complete!")
		return nil
	},
}

// findRepository tries to find the repository in common locations
func findRepository() (string, error) {
	// First, try to find it relative to the current binary
	currentBinary, err := os.Executable()
	if err != nil {
		return "", err
	}

	// Resolve symlinks
	realPath, err := filepath.EvalSymlinks(currentBinary)
	if err == nil {
		currentBinary = realPath
	}

	// Check if we're already in a git repository (current working directory)
	if cwd, err := os.Getwd(); err == nil {
		if isGitRepo(cwd) {
			return cwd, nil
		}
	}

	// Check common repository locations relative to binary
	// If installed in ~/.local/bin, repo might be in ~/Projects/cursor-chat-cli
	binaryDir := filepath.Dir(currentBinary)
	
	// Common patterns:
	// ~/.local/bin/cursor-session -> ~/Projects/cursor-chat-cli
	// /usr/local/bin/cursor-session -> ~/Projects/cursor-chat-cli
	homeDir, err := os.UserHomeDir()
	if err == nil {
		commonPaths := []string{
			filepath.Join(homeDir, "Projects", "cursor-chat-cli"),
			filepath.Join(homeDir, "projects", "cursor-chat-cli"),
			filepath.Join(homeDir, "Code", "cursor-chat-cli"),
			filepath.Join(homeDir, "code", "cursor-chat-cli"),
			filepath.Join(homeDir, "go", "src", "github.com", "k", "cursor-session"),
			filepath.Join(homeDir, "go", "pkg", "mod", "github.com", "k", "cursor-session@*"),
		}

		for _, path := range commonPaths {
			if isGitRepo(path) {
				return path, nil
			}
		}
	}

	// Try to find it by walking up from the binary location
	// (in case it's in a subdirectory of the repo)
	dir := binaryDir
	for i := 0; i < 10; i++ { // Limit depth
		if isGitRepo(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("could not find repository")
}

// isGitRepo checks if a directory is a git repository
func isGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy file contents
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		n, err := sourceFile.Read(buf)
		if n > 0 {
			if _, writeErr := destFile.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}

