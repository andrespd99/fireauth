package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cashea-bnpl/auth-devtools/internal/logger"
)

const (
	repoOwner = "cashea-bnpl"
	repoName  = "auth-devtools"
	binaryName = "cashea-auth"
)

// GitHubAPIBase is the base URL for GitHub API calls. Overridable for testing.
var GitHubAPIBase = "https://api.github.com"

// GitHubRelease represents a GitHub release.
type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GitHubAsset `json:"assets"`
}

// GitHubAsset represents a release asset.
type GitHubAsset struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ResolveToken returns a GitHub token from the environment or gh CLI.
func ResolveToken() (string, error) {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		logger.Debug("using GITHUB_TOKEN from environment")
		return token, nil
	}

	// Try gh CLI.
	out, err := exec.Command("gh", "auth", "token").Output()
	if err == nil {
		token := strings.TrimSpace(string(out))
		if token != "" {
			logger.Debug("using token from gh CLI")
			return token, nil
		}
	}

	return "", fmt.Errorf("GITHUB_TOKEN is required (private repo)\n\n  Option 1: export GITHUB_TOKEN=ghp_...\n  Option 2: install gh CLI and run 'gh auth login'")
}

// FetchLatestRelease fetches the latest release from GitHub.
func FetchLatestRelease(token string) (*GitHubRelease, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", GitHubAPIBase, repoOwner, repoName)
	logger.Debug("fetching latest release", "url", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	logger.Debug("release API response", "status", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("no releases found — push a tag first (git tag v0.1.0 && git push origin v0.1.0)")
		}
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("GitHub token is invalid or lacks repo access (HTTP %d)", resp.StatusCode)
		}
		return nil, fmt.Errorf("GitHub API error: HTTP %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("parsing release response: %w", err)
	}

	logger.Debug("latest release", "tag", release.TagName, "assets", len(release.Assets))
	return &release, nil
}

// FindAsset finds the matching asset for the current OS/arch.
func FindAsset(release *GitHubRelease, goos, goarch string) (*GitHubAsset, error) {
	target := fmt.Sprintf("%s_%s_%s.tar.gz", binaryName, goos, goarch)
	logger.Debug("looking for asset", "name", target)

	for _, a := range release.Assets {
		if a.Name == target {
			logger.Debug("asset found", "id", a.ID, "name", a.Name)
			return &a, nil
		}
	}

	available := make([]string, len(release.Assets))
	for i, a := range release.Assets {
		available[i] = a.Name
	}
	return nil, fmt.Errorf("no release asset for %s/%s (looking for %s)\navailable: %s",
		goos, goarch, target, strings.Join(available, ", "))
}

// DownloadAsset downloads a release asset by ID. For private repos this uses
// the GitHub API with Accept: application/octet-stream.
func DownloadAsset(token string, assetID int) ([]byte, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/assets/%d", GitHubAPIBase, repoOwner, repoName, assetID)
	logger.Debug("downloading asset", "url", url, "asset_id", assetID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading asset: %w", err)
	}
	defer resp.Body.Close()

	logger.Debug("download response", "status", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading download: %w", err)
	}

	logger.Debug("asset downloaded", "bytes", len(data))
	return data, nil
}

// ExtractBinary extracts the binary from a tar.gz archive.
func ExtractBinary(archive []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("decompressing archive: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading tar: %w", err)
		}

		if filepath.Base(header.Name) == binaryName && header.Typeflag == tar.TypeReg {
			logger.Debug("found binary in archive", "name", header.Name, "size", header.Size)
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("extracting binary: %w", err)
			}
			return data, nil
		}
	}

	return nil, fmt.Errorf("binary %q not found in archive", binaryName)
}

// Apply replaces the current binary with the new one using atomic rename.
func Apply(binaryData []byte) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary path: %w", err)
	}

	// Resolve symlinks to get the real path.
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolving binary path: %w", err)
	}
	logger.Debug("replacing binary", "path", execPath)

	// Get current permissions.
	info, err := os.Stat(execPath)
	if err != nil {
		return fmt.Errorf("stat current binary: %w", err)
	}
	perm := info.Mode().Perm()

	// Write new binary to temp file in the same directory (for atomic rename).
	dir := filepath.Dir(execPath)
	tmp, err := os.CreateTemp(dir, binaryName+"-update-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	logger.Debug("writing new binary to temp file", "path", tmpPath)

	if _, err := tmp.Write(binaryData); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing new binary: %w", err)
	}
	tmp.Close()

	if err := os.Chmod(tmpPath, perm); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("setting permissions: %w", err)
	}

	// Atomic rename.
	if err := os.Rename(tmpPath, execPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("replacing binary: %w", err)
	}

	logger.Debug("binary replaced successfully")
	return nil
}

// NeedsUpdate compares the current version with the latest release tag.
// Returns true if the versions differ.
func NeedsUpdate(currentVersion, latestTag string) bool {
	// Strip leading "v" for comparison.
	current := strings.TrimPrefix(currentVersion, "v")
	latest := strings.TrimPrefix(latestTag, "v")
	return current != latest
}

// CurrentPlatform returns the current OS and architecture.
func CurrentPlatform() (string, string) {
	return runtime.GOOS, runtime.GOARCH
}
