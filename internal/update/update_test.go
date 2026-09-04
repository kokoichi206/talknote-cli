package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssetName(t *testing.T) {
	cases := map[string]string{
		"darwin/arm64":  "talknote-cli_0.1.0_darwin_arm64.tar.gz",
		"linux/amd64":   "talknote-cli_0.1.0_linux_amd64.tar.gz",
		"windows/amd64": "talknote-cli_0.1.0_windows_amd64.zip",
	}
	got := map[string]string{
		"darwin/arm64":  AssetName("0.1.0", "darwin", "arm64"),
		"linux/amd64":   AssetName("0.1.0", "linux", "amd64"),
		"windows/amd64": AssetName("0.1.0", "windows", "amd64"),
	}
	for k, want := range cases {
		if got[k] != want {
			t.Errorf("AssetName(%s) = %q, want %q", k, got[k], want)
		}
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"1.0.1", "1.0.0", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "1.1.0", false},
		{"1.0.0", "dev", true}, // 開発ビルドは常に更新対象
	}
	for _, c := range cases {
		got, err := isNewer(c.latest, c.current)
		if err != nil {
			t.Errorf("isNewer(%q,%q): %v", c.latest, c.current, err)
			continue
		}
		if got != c.want {
			t.Errorf("isNewer(%q,%q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestRunDryRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"tag_name":"v1.2.0","html_url":"https://example/rel","assets":[{"name":"talknote-cli_1.2.0_darwin_arm64.tar.gz","browser_download_url":"https://example/a"}]}`)
	}))
	defer srv.Close()

	res, err := Run(context.Background(), srv.Client(), Options{
		CurrentVersion: "1.0.0",
		ExecutablePath: "/nonexistent/tn",
		GOOS:           "darwin",
		GOARCH:         "arm64",
		DryRun:         true,
		releasesAPI:    srv.URL,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.UpdateAvailable || res.Updated {
		t.Errorf("dry-run result = %+v", res)
	}
	if res.LatestVersion != "1.2.0" || res.AssetName != "talknote-cli_1.2.0_darwin_arm64.tar.gz" {
		t.Errorf("result = %+v", res)
	}
}

func TestRunReplacesExecutableAfterChecksumVerification(t *testing.T) {
	archive := tarGzWithBinary(t, []byte("new binary"))
	digest := sha256.Sum256(archive)
	assetName := "talknote-cli_1.1.0_darwin_arm64.tar.gz"

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/release":
			fmt.Fprintf(w, `{"tag_name":"v1.1.0","html_url":"https://example/rel","assets":[{"name":%q,"browser_download_url":%q},{"name":"checksums.txt","browser_download_url":%q}]}`, assetName, server.URL+"/archive", server.URL+"/checksums")
		case "/archive":
			_, _ = w.Write(archive)
		case "/checksums":
			fmt.Fprintf(w, "%s  %s\n", hex.EncodeToString(digest[:]), assetName)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	executable := filepath.Join(t.TempDir(), "tn")
	if err := os.WriteFile(executable, []byte("old binary"), 0o751); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(executable, 0o751); err != nil {
		t.Fatal(err)
	}
	result, err := Run(context.Background(), server.Client(), Options{
		CurrentVersion: "1.0.0",
		ExecutablePath: executable,
		GOOS:           "darwin",
		GOARCH:         "arm64",
		releasesAPI:    server.URL + "/release",
	})
	if err != nil || !result.Updated {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	data, err := os.ReadFile(executable)
	if err != nil || string(data) != "new binary" {
		t.Fatalf("data=%q err=%v", data, err)
	}
	info, err := os.Stat(executable)
	if err != nil || info.Mode().Perm() != 0o751 {
		t.Fatalf("mode=%v err=%v", info.Mode().Perm(), err)
	}
}

func TestRunChecksumMismatchKeepsExecutable(t *testing.T) {
	archive := tarGzWithBinary(t, []byte("new binary"))
	assetName := "talknote-cli_1.1.0_darwin_arm64.tar.gz"

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/release":
			fmt.Fprintf(w, `{"tag_name":"v1.1.0","html_url":"https://example/rel","assets":[{"name":%q,"browser_download_url":%q},{"name":"checksums.txt","browser_download_url":%q}]}`, assetName, server.URL+"/archive", server.URL+"/checksums")
		case "/archive":
			_, _ = w.Write(archive)
		case "/checksums":
			fmt.Fprintf(w, "%064d  %s\n", 0, assetName)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	executable := filepath.Join(t.TempDir(), "tn")
	if err := os.WriteFile(executable, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Run(context.Background(), server.Client(), Options{
		CurrentVersion: "1.0.0",
		ExecutablePath: executable,
		GOOS:           "darwin",
		GOARCH:         "arm64",
		releasesAPI:    server.URL + "/release",
	})
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err=%v", err)
	}
	data, readErr := os.ReadFile(executable)
	if readErr != nil || string(data) != "old binary" {
		t.Fatalf("data=%q err=%v", data, readErr)
	}
}

func tarGzWithBinary(t *testing.T, content []byte) []byte {
	t.Helper()
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "tn", Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}
