package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/andrespd99/fireauth/internal/logger"
	"golang.org/x/mod/semver"
)

// httpClient is used for all GitHub API calls. A timeout prevents the CLI
// from hanging indefinitely if GitHub is unreachable.
var httpClient = &http.Client{Timeout: 60 * time.Second}

const (
	repoOwner  = "andrespd99"
	repoName   = "fireauth"
	binaryName = "fireauth"
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

// ResolveToken returns a GitHub token from the environment or gh CLI. If no
// token is found, an empty string is returned — the caller proceeds
// unauthenticated, which works for public repos subject to GitHub's rate
// limits.
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

	// No token found — proceed unauthenticated (works for public repos with
	// rate limits).
	logger.Debug("no GitHub token found, proceeding unauthenticated")
	return "", nil
}

// FetchLatestRelease fetches the latest release from GitHub.
func FetchLatestRelease(ctx context.Context, token string) (*GitHubRelease, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", GitHubAPIBase, repoOwner, repoName)
	logger.Debug("fetching latest release", "url", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "fireauth/updater")

	resp, err := httpClient.Do(req)
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
			return nil, fmt.Errorf("GitHub API returned HTTP %d — if you are rate limited, set GITHUB_TOKEN or run 'gh auth login'", resp.StatusCode)
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
func DownloadAsset(ctx context.Context, token string, assetID int) ([]byte, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/assets/%d", GitHubAPIBase, repoOwner, repoName, assetID)
	logger.Debug("downloading asset", "url", url, "asset_id", assetID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("User-Agent", "fireauth/updater")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading asset: %w", err)
	}
	defer resp.Body.Close()

	logger.Debug("download response", "status", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 100<<20))
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

// NeedsUpdate compares the current version with the latest release tag using
// proper semver comparison. Returns true if the latest version is newer.
func NeedsUpdate(currentVersion, latestTag string) bool {
	current := normalizeVersion(currentVersion)
	latest := normalizeVersion(latestTag)
	// If current is not a valid semver (e.g. "dev"), always offer update.
	if !semver.IsValid(current) {
		return true
	}
	if !semver.IsValid(latest) {
		return false
	}
	return semver.Compare(latest, current) > 0
}

// normalizeVersion ensures a leading "v" prefix and strips the non-standard
// "-stable" suffix so that "0.3.0-stable" and "0.3.0" compare as equal.
func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if !strings.HasPrefix(v, "v") && !strings.HasPrefix(v, "V") {
		v = "v" + v
	}
	v = strings.TrimSuffix(v, "-stable")
	return v
}

// FindChecksumAsset finds the checksums.txt asset in a release.
func FindChecksumAsset(release *GitHubRelease) (*GitHubAsset, error) {
	for _, a := range release.Assets {
		if a.Name == "checksums.txt" {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("checksums.txt not found in release assets")
}

// VerifyChecksum downloads checksums.txt, finds the expected hash for the
// given archive name, and verifies the downloaded archive data matches.
func VerifyChecksum(ctx context.Context, token string, checksumAssetID int, archiveName string, archiveData []byte) error {
	checksumData, err := DownloadAsset(ctx, token, checksumAssetID)
	if err != nil {
		return fmt.Errorf("downloading checksums: %w", err)
	}

	expectedHash, err := parseChecksum(checksumData, archiveName)
	if err != nil {
		return err
	}

	actualHash := sha256.Sum256(archiveData)
	actual := hex.EncodeToString(actualHash[:])

	if actual != expectedHash {
		return fmt.Errorf("checksum mismatch for %s\n  expected: %s\n  actual:   %s", archiveName, expectedHash, actual)
	}

	logger.Debug("checksum verified", "archive", archiveName, "hash", actual)
	return nil
}

// parseChecksum parses a checksums file (format: "hash  filename" per line)
// and returns the hash for the given filename.
func parseChecksum(data []byte, filename string) (string, error) {
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == filename {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("no checksum found for %s in checksums.txt", filename)
}

// CurrentPlatform returns the current OS and architecture.
func CurrentPlatform() (string, string) {
	return runtime.GOOS, runtime.GOARCH
}
