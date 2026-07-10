package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchLatestRelease_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token test-token" {
			t.Errorf("auth header = %q", r.Header.Get("Authorization"))
		}

		release := GitHubRelease{
			TagName: "v0.2.0",
			Assets: []GitHubAsset{
				{ID: 1, Name: "fireauth_darwin_arm64.tar.gz"},
				{ID: 2, Name: "fireauth_linux_amd64.tar.gz"},
				{ID: 3, Name: "checksums.txt"},
			},
		}
		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	orig := GitHubAPIBase
	GitHubAPIBase = server.URL
	defer func() { GitHubAPIBase = orig }()

	release, err := FetchLatestRelease("test-token")
	if err != nil {
		t.Fatalf("FetchLatestRelease: %v", err)
	}
	if release.TagName != "v0.2.0" {
		t.Errorf("TagName = %q, want v0.2.0", release.TagName)
	}
	if len(release.Assets) != 3 {
		t.Errorf("Assets count = %d, want 3", len(release.Assets))
	}
}

func TestFetchLatestRelease_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	orig := GitHubAPIBase
	GitHubAPIBase = server.URL
	defer func() { GitHubAPIBase = orig }()

	_, err := FetchLatestRelease("test-token")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestFetchLatestRelease_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	orig := GitHubAPIBase
	GitHubAPIBase = server.URL
	defer func() { GitHubAPIBase = orig }()

	_, err := FetchLatestRelease("bad-token")
	if err == nil {
		t.Fatal("expected error for 401")
	}
}

func TestFindAsset(t *testing.T) {
	release := &GitHubRelease{
		TagName: "v0.2.0",
		Assets: []GitHubAsset{
			{ID: 1, Name: "fireauth_darwin_arm64.tar.gz"},
			{ID: 2, Name: "fireauth_linux_amd64.tar.gz"},
			{ID: 3, Name: "fireauth_darwin_amd64.tar.gz"},
			{ID: 4, Name: "checksums.txt"},
		},
	}

	asset, err := FindAsset(release, "darwin", "arm64")
	if err != nil {
		t.Fatalf("FindAsset: %v", err)
	}
	if asset.ID != 1 {
		t.Errorf("asset ID = %d, want 1", asset.ID)
	}

	asset, err = FindAsset(release, "linux", "amd64")
	if err != nil {
		t.Fatalf("FindAsset linux: %v", err)
	}
	if asset.ID != 2 {
		t.Errorf("asset ID = %d, want 2", asset.ID)
	}
}

func TestFindAsset_NotFound(t *testing.T) {
	release := &GitHubRelease{
		TagName: "v0.2.0",
		Assets: []GitHubAsset{
			{ID: 1, Name: "fireauth_darwin_arm64.tar.gz"},
		},
	}

	_, err := FindAsset(release, "windows", "amd64")
	if err == nil {
		t.Fatal("expected error for missing platform")
	}
}

func TestDownloadAsset_Success(t *testing.T) {
	expectedData := []byte("fake-binary-content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/octet-stream" {
			t.Errorf("Accept header = %q", r.Header.Get("Accept"))
		}
		w.Write(expectedData)
	}))
	defer server.Close()

	orig := GitHubAPIBase
	GitHubAPIBase = server.URL
	defer func() { GitHubAPIBase = orig }()

	data, err := DownloadAsset("test-token", 123)
	if err != nil {
		t.Fatalf("DownloadAsset: %v", err)
	}
	if !bytes.Equal(data, expectedData) {
		t.Error("downloaded data mismatch")
	}
}

func TestDownloadAsset_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	orig := GitHubAPIBase
	GitHubAPIBase = server.URL
	defer func() { GitHubAPIBase = orig }()

	_, err := DownloadAsset("test-token", 123)
	if err == nil {
		t.Fatal("expected error for 403")
	}
}

func TestNeedsUpdate(t *testing.T) {
	tests := []struct {
		current, latest string
		want            bool
	}{
		{"v0.1.0", "v0.2.0", true},
		{"0.1.0", "v0.1.0", false},
		{"v0.2.0", "v0.2.0", false},
		{"dev", "v0.1.0", true},
		{"v0.1.0", "v0.1.0", false},
	}
	for _, tt := range tests {
		got := NeedsUpdate(tt.current, tt.latest)
		if got != tt.want {
			t.Errorf("NeedsUpdate(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
		}
	}
}

func TestExtractBinary(t *testing.T) {
	// Build a tar.gz archive with a fake binary inside.
	binaryContent := []byte("#!/bin/sh\necho hello")
	archive := createTarGz(t, "fireauth", binaryContent)

	extracted, err := ExtractBinary(archive)
	if err != nil {
		t.Fatalf("ExtractBinary: %v", err)
	}
	if !bytes.Equal(extracted, binaryContent) {
		t.Errorf("extracted content mismatch")
	}
}

func TestExtractBinary_NotFound(t *testing.T) {
	archive := createTarGz(t, "wrong-name", []byte("data"))

	_, err := ExtractBinary(archive)
	if err == nil {
		t.Fatal("expected error when binary not in archive")
	}
}

// createTarGz creates a tar.gz archive with a single file.
func createTarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name: name,
		Size: int64(len(content)),
		Mode: 0755,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}

	tw.Close()
	gw.Close()
	return buf.Bytes()
}
