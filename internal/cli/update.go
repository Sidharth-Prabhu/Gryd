package cli

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	githubOwner    = "Sidharth-Prabhu"
	githubRepo     = "gryd"
	cacheFileName  = "version_check.json"
	checkInterval  = 6 * time.Hour
	requestTimeout = 3 * time.Second
)

// releaseInfo holds GitHub release metadata
type releaseInfo struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

type versionCache struct {
	LastChecked   time.Time `json:"last_checked"`
	LatestVersion string    `json:"latest_version"`
	ReleaseURL    string    `json:"release_url"`
}

func getCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	cacheDir := filepath.Join(home, ".gryd")
	_ = os.MkdirAll(cacheDir, 0755)
	return filepath.Join(cacheDir, cacheFileName)
}

// FetchLatestRelease checks GitHub API for the latest release.
func FetchLatestRelease(timeout time.Duration) (*releaseInfo, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)
	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "gryd-cli/"+Version)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var rel releaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// CheckForUpdateCached checks if a new version is available without delaying CLI commands.
// It uses a local cache file that updates periodically in the background.
func CheckForUpdateCached() (string, bool) {
	cachePath := getCachePath()
	if cachePath == "" {
		return "", false
	}

	var cache versionCache
	if data, err := os.ReadFile(cachePath); err == nil {
		_ = json.Unmarshal(data, &cache)
	}

	// If cache is older than checkInterval, refresh in background
	if time.Since(cache.LastChecked) > checkInterval {
		go func() {
			rel, err := FetchLatestRelease(requestTimeout)
			if err == nil && rel.TagName != "" {
				newCache := versionCache{
					LastChecked:   time.Now(),
					LatestVersion: rel.TagName,
					ReleaseURL:    rel.HTMLURL,
				}
				if b, err := json.Marshal(newCache); err == nil {
					_ = os.WriteFile(cachePath, b, 0644)
				}
			}
		}()
	}

	// Compare current Version with cached latest version
	if cache.LatestVersion != "" && isNewerVersion(cache.LatestVersion, Version) {
		return cache.LatestVersion, true
	}

	return "", false
}

// isInstalledViaHomebrew checks if the running binary resides within a Homebrew cellar/prefix
func isInstalledViaHomebrew(execPath string) bool {
	lower := strings.ToLower(execPath)
	return strings.Contains(lower, "/cellar/") ||
		strings.Contains(lower, "/homebrew/") ||
		strings.Contains(lower, "/opt/homebrew/") ||
		strings.Contains(lower, "/usr/local/cellar/")
}

// CheckUpdatePrompt prints an update notice if a newer release exists.
func CheckUpdatePrompt() {
	if latest, available := CheckForUpdateCached(); available {
		execPath, _ := os.Executable()
		if evalPath, err := filepath.EvalSymlinks(execPath); err == nil {
			execPath = evalPath
		}

		fmt.Fprintf(os.Stderr, "\n\033[1;33m⚡ A new version of Gryd is available: \033[1;32m%s\033[1;33m (current: \033[1;30m%s\033[1;33m)\033[0m\n", latest, Version)
		if isInstalledViaHomebrew(execPath) {
			fmt.Fprintf(os.Stderr, "\033[1;33m   Run \033[1;36mbrew upgrade gryd\033[1;33m (or \033[1;36mgryd update\033[1;33m) to upgrade.\033[0m\n\n")
		} else {
			fmt.Fprintf(os.Stderr, "\033[1;33m   Run \033[1;36mgryd update\033[1;33m to automatically upgrade.\033[0m\n\n")
		}
	}
}

// isNewerVersion compares semver strings (e.g. "v0.2.0" vs "0.1.0-dev")
func isNewerVersion(latest, current string) bool {
	cleanLatest := strings.TrimPrefix(strings.TrimSpace(latest), "v")
	cleanCurrent := strings.TrimPrefix(strings.TrimSpace(current), "v")

	// If current is dev, any released version is newer
	if strings.Contains(cleanCurrent, "dev") {
		return true
	}

	return cleanLatest != cleanCurrent && cleanLatest > cleanCurrent
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for and install the latest Gryd release",
	Long:  `Downloads and replaces the currently running gryd binary with the latest GitHub release.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🔍 Checking for updates...")
		rel, err := FetchLatestRelease(10 * time.Second)
		if err != nil {
			return fmt.Errorf("failed to fetch release information: %w", err)
		}

		cleanLatest := strings.TrimPrefix(rel.TagName, "v")
		cleanCurrent := strings.TrimPrefix(Version, "v")
		_ = cleanLatest
		_ = cleanCurrent

		if !isNewerVersion(rel.TagName, Version) {
			fmt.Printf("✓ Gryd is already up to date! (version %s)\n", Version)
			return nil
		}

		fmt.Printf("🚀 Found newer version: %s (current: %s)\n", rel.TagName, Version)

		// Determine target asset name pattern
		goos := runtime.GOOS
		goarch := runtime.GOARCH

		var downloadURL, assetName string
		for _, asset := range rel.Assets {
			nameLower := strings.ToLower(asset.Name)
			if strings.Contains(nameLower, goos) && strings.Contains(nameLower, goarch) {
				downloadURL = asset.BrowserDownloadURL
				assetName = asset.Name
				break
			}
		}

		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to locate current executable path: %w", err)
		}
		// Resolve symlinks
		if evalPath, err := filepath.EvalSymlinks(execPath); err == nil {
			execPath = evalPath
		}

		if isInstalledViaHomebrew(execPath) {
			fmt.Printf("💡 Gryd was detected as installed via Homebrew (%s).\n", execPath)
			fmt.Println("To upgrade via Homebrew, run:")
			fmt.Println("  \033[1;36mbrew update && brew upgrade gryd\033[0m")
			fmt.Print("\nDo you still want to directly overwrite this binary? (y/N): ")
			var answer string
			_, _ = fmt.Scanln(&answer)
			if strings.ToLower(strings.TrimSpace(answer)) != "y" {
				fmt.Println("Update cancelled. Please use 'brew upgrade gryd'.")
				return nil
			}
		}

		if downloadURL == "" {
			// No prebuilt binary found for architecture, offer go install fallback
			fmt.Println("No pre-built archive asset found for your architecture.")
			fmt.Printf("You can upgrade via Go by running:\n  go install github.com/loom-cli/gryd/cmd/gryd@%s\n", rel.TagName)
			return nil
		}

		fmt.Printf("📦 Downloading %s from %s...\n", assetName, downloadURL)
		tmpDir, err := os.MkdirTemp("", "gryd-update")
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		archivePath := filepath.Join(tmpDir, assetName)
		if err := downloadFile(downloadURL, archivePath); err != nil {
			return fmt.Errorf("download failed: %w", err)
		}

		// Extract binary from archive
		extractedBinary := filepath.Join(tmpDir, "gryd_new")
		if strings.HasSuffix(assetName, ".tar.gz") || strings.HasSuffix(assetName, ".tgz") {
			if err := extractTarGz(archivePath, "gryd", extractedBinary); err != nil {
				return fmt.Errorf("failed to extract tar.gz: %w", err)
			}
		} else if strings.HasSuffix(assetName, ".zip") {
			if err := extractZip(archivePath, "gryd", extractedBinary); err != nil {
				return fmt.Errorf("failed to extract zip: %w", err)
			}
		} else {
			// Raw binary
			extractedBinary = archivePath
		}

		// Verify binary runs and test permission
		_ = os.Chmod(extractedBinary, 0755)

		// Replace current binary atomically
		fmt.Printf("🔄 Installing to %s...\n", execPath)
		oldExec := execPath + ".old"
		_ = os.Remove(oldExec)

		if err := os.Rename(execPath, oldExec); err != nil {
			return fmt.Errorf("permission denied writing to %s (try running with sudo or check directory permissions): %w", execPath, err)
		}

		if err := copyExecutableFile(extractedBinary, execPath); err != nil {
			_ = os.Rename(oldExec, execPath) // Rollback
			return fmt.Errorf("failed to install new binary: %w", err)
		}
		_ = os.Remove(oldExec)

		// Update cache
		if cachePath := getCachePath(); cachePath != "" {
			cacheData, _ := json.Marshal(versionCache{
				LastChecked:   time.Now(),
				LatestVersion: rel.TagName,
				ReleaseURL:    rel.HTMLURL,
			})
			_ = os.WriteFile(cachePath, cacheData, 0644)
		}

		fmt.Printf("\n\033[1;32m✓ Successfully updated Gryd to %s!\033[0m\n", rel.TagName)
		fmt.Printf("Run 'gryd --version' to check.\n")
		return nil
	},
}

func downloadFile(url, destPath string) error {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error %d downloading asset", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractTarGz(archivePath, binName, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		baseName := filepath.Base(header.Name)
		if baseName == binName || baseName == binName+".exe" {
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr)
			out.Close()
			return copyErr
		}
	}
	return fmt.Errorf("binary '%s' not found inside archive", binName)
}

func extractZip(archivePath, binName, destPath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		baseName := filepath.Base(f.Name)
		if baseName == binName || baseName == binName+".exe" {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			defer out.Close()

			_, copyErr := io.Copy(out, rc)
			return copyErr
		}
	}
	return fmt.Errorf("binary '%s' not found inside zip archive", binName)
}

func copyExecutableFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return os.Chmod(dst, 0755)
}
