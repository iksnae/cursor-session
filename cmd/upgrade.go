package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/iksnae/cursor-session/internal"
	"github.com/spf13/cobra"
)

const (
	repoOwner = "iksnae"
	repoName  = "cursor-session"
	repoURL   = "https://api.github.com/repos/" + repoOwner + "/" + repoName
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade cursor-session to the latest version",
	Long: `Upgrade cursor-session to the latest released version from GitHub.

This command will:
1. Check the current installed version
2. Fetch the latest release from GitHub
3. Download and install the latest binary if a newer version is available

If you installed via 'go install', you can also upgrade by running:
  go install github.com/iksnae/cursor-session@latest`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Get current version
		currentVersion, err := parseCurrentVersion()
		if err != nil {
			internal.LogWarn("Could not parse current version: %v", err)
			currentVersion = nil
		}

		// Use a struct to pass data between steps
		type upgradeData struct {
			latestRelease  *githubRelease
			latestVersion  *semver.Version
			binaryPath     string
			tempDir        string
		}
		data := &upgradeData{}

		// Fetch latest release
		steps := []internal.ProgressStep{
			{
				Message: "Checking for latest version",
				Fn: func() error {
					latestRelease, err := fetchLatestRelease()
					if err != nil {
						return fmt.Errorf("failed to fetch latest release: %w", err)
					}

					latestVersion, err := semver.NewVersion(strings.TrimPrefix(latestRelease.TagName, "v"))
					if err != nil {
						return fmt.Errorf("failed to parse latest version: %w", err)
					}

					// Check if upgrade is needed
					if currentVersion != nil && !latestVersion.GreaterThan(currentVersion) {
						internal.PrintSuccess(fmt.Sprintf("You are already on the latest version: %s", latestRelease.TagName))
						return fmt.Errorf("already up to date") //nolint:stylecheck // This is a controlled exit
					}

					// Store latest release info for next step
					data.latestRelease = latestRelease
					data.latestVersion = latestVersion

					if currentVersion != nil {
						fmt.Printf("Current version: %s\n", currentVersion.String())
					}
					fmt.Printf("Latest version: %s\n", latestVersion.String())

					return nil
				},
			},
			{
				Message: "Downloading latest release",
				Fn: func() error {
					if data.latestRelease == nil {
						return fmt.Errorf("latest release not found")
					}

					// Get download URL for current platform
					downloadURL, err := getDownloadURL(data.latestRelease.TagName)
					if err != nil {
						return fmt.Errorf("failed to get download URL: %w", err)
					}

					// Download the binary
					tempDir, err := os.MkdirTemp("", "cursor-session-upgrade-*")
					if err != nil {
						return fmt.Errorf("failed to create temp directory: %w", err)
					}
					data.tempDir = tempDir

					archivePath := filepath.Join(tempDir, "cursor-session.tar.gz")
					if err := downloadFile(downloadURL, archivePath); err != nil {
						return fmt.Errorf("failed to download binary: %w", err)
					}

					// Extract binary
					binaryPath := filepath.Join(tempDir, "cursor-session")
					if err := extractBinary(archivePath, binaryPath); err != nil {
						return fmt.Errorf("failed to extract binary: %w", err)
					}

					// Store binary path for next step
					data.binaryPath = binaryPath

					return nil
				},
			},
			{
				Message: "Installing new version",
				Fn: func() error {
					if data.binaryPath == "" {
						return fmt.Errorf("binary path not found")
					}

					// Get current binary path
					currentBinary, err := os.Executable()
					if err != nil {
						return fmt.Errorf("failed to get current binary path: %w", err)
					}

					// Resolve symlinks
					realPath, err := filepath.EvalSymlinks(currentBinary)
					if err == nil {
						currentBinary = realPath
					}

					// Make sure target directory exists
					targetDir := filepath.Dir(currentBinary)
					if err := os.MkdirAll(targetDir, 0755); err != nil {
						return fmt.Errorf("failed to create target directory: %w", err)
					}

					// Copy new binary to target location
					if err := copyFile(data.binaryPath, currentBinary); err != nil {
						return fmt.Errorf("failed to install binary: %w", err)
					}

					// Make it executable
					if err := os.Chmod(currentBinary, 0755); err != nil {
						return fmt.Errorf("failed to make binary executable: %w", err)
					}

					return nil
				},
			},
			{
				Message: "Verifying installation",
				Fn: func() error {
					currentBinary, err := os.Executable()
					if err != nil {
						return fmt.Errorf("failed to get binary path: %w", err)
					}

					verifyCmd := exec.Command(currentBinary, "--version")
					output, err := verifyCmd.Output()
					if err != nil {
						internal.LogWarn("Installation completed but verification failed: %v", err)
						return nil // Don't fail on verification
					}
					fmt.Println()
					fmt.Println("Installed version:")
					fmt.Print(string(output))
					return nil
				},
			},
		}

		if err := internal.ShowProgressWithSteps(ctx, steps); err != nil {
			// Cleanup temp directory
			if data.tempDir != "" {
				os.RemoveAll(data.tempDir)
			}
			// Check if it's the "already up to date" error
			if strings.Contains(err.Error(), "already up to date") {
				return nil
			}
			return err
		}

		// Cleanup temp directory
		if data.tempDir != "" {
			os.RemoveAll(data.tempDir)
		}

		internal.PrintSuccess("Upgrade complete!")
		return nil
	},
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func parseCurrentVersion() (*semver.Version, error) {
	// The version string format is: "version (commit: commit, built: date)"
	// Extract just the version part
	versionStr := version
	if versionStr == "dev" {
		return nil, fmt.Errorf("running development version")
	}

	// Remove 'v' prefix if present
	versionStr = strings.TrimPrefix(versionStr, "v")

	// Parse version
	v, err := semver.NewVersion(versionStr)
	if err != nil {
		return nil, fmt.Errorf("invalid version format: %w", err)
	}

	return v, nil
}

func fetchLatestRelease() (*githubRelease, error) {
	url := repoURL + "/releases/latest"
	resp, err := http.Get(url) //nolint:gosec // GitHub API URL is safe
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch release info: status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	return &release, nil
}

func getDownloadURL(tagName string) (string, error) {
	// Determine OS and architecture
	osName := runtime.GOOS
	archName := runtime.GOARCH

	// Map architecture names
	if archName == "amd64" {
		archName = "amd64"
	} else if archName == "arm64" {
		archName = "arm64"
	} else {
		return "", fmt.Errorf("unsupported architecture: %s", archName)
	}

	// Map OS names
	if osName == "darwin" {
		osName = "darwin"
	} else if osName == "linux" {
		osName = "linux"
	} else {
		return "", fmt.Errorf("unsupported OS: %s", osName)
	}

	// Construct download URL
	versionWithoutV := strings.TrimPrefix(tagName, "v")
	downloadURL := fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/%s/cursor-session-%s-%s-%s.tar.gz",
		repoOwner, repoName, tagName, versionWithoutV, osName, archName,
	)

	return downloadURL, nil
}

func downloadFile(url, destPath string) error {
	resp, err := http.Get(url) //nolint:gosec // URL is constructed from known GitHub releases
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: status %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func extractBinary(archivePath, destPath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		if header.Typeflag == tar.TypeReg && filepath.Base(header.Name) == "cursor-session" {
			out, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("failed to create binary file: %w", err)
			}
			defer out.Close()

			if _, err := io.Copy(out, tr); err != nil {
				return fmt.Errorf("failed to extract binary: %w", err)
			}

			// Make it executable
			if err := os.Chmod(destPath, 0755); err != nil {
				return fmt.Errorf("failed to make binary executable: %w", err)
			}

			return nil
		}
	}

	return fmt.Errorf("binary not found in archive")
}

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

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return nil
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}

