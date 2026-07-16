package updater

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestClientCheckFindsNewerRelease(t *testing.T) {
	t.Parallel()
	body := `{
      "tag_name":"v1.2.0",
      "draft":false,
      "prerelease":false,
      "assets":[
        {"name":"rvmm_macOS_arm64.pkg","browser_download_url":"https://github.com/rxtech-lab/macos-github-action-vm/releases/download/v1.2.0/rvmm_macOS_arm64.pkg","size":42},
        {"name":"rvmm_macOS_arm64.pkg.sha256","browser_download_url":"https://github.com/rxtech-lab/macos-github-action-vm/releases/download/v1.2.0/rvmm_macOS_arm64.pkg.sha256","size":80}
      ]
    }`
	client := NewClient("v1.1.0")
	client.HTTPClient.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(http.StatusOK, body), nil
	})

	update, err := client.Check(context.Background(), "v1.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if !update.Available || update.LatestVersion != "v1.2.0" {
		t.Fatalf("unexpected update: %+v", update)
	}
}

func TestClientCheckRequiresChecksumAsset(t *testing.T) {
	t.Parallel()
	body := `{"tag_name":"v1.2.0","assets":[{"name":"rvmm_macOS_arm64.pkg","browser_download_url":"https://github.com/example.pkg","size":42}]}`
	client := NewClient("v1.1.0")
	client.HTTPClient.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(http.StatusOK, body), nil
	})
	if _, err := client.Check(context.Background(), "v1.1.0"); err == nil || !strings.Contains(err.Error(), ChecksumAssetName) {
		t.Fatalf("Check() error = %v, want missing checksum", err)
	}
}

func TestClientDownloadVerifiesChecksum(t *testing.T) {
	t.Parallel()
	packageBody := "signed package contents"
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(packageBody)))
	client := NewClient("v1.1.0")
	client.HTTPClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if strings.HasSuffix(request.URL.Path, ".sha256") {
			return response(http.StatusOK, digest+"  "+PackageAssetName+"\n"), nil
		}
		return response(http.StatusOK, packageBody), nil
	})
	update := Update{
		Available: true,
		Package:   ReleaseAsset{Name: PackageAssetName, BrowserDownloadURL: "https://github.com/package", Size: int64(len(packageBody))},
		Checksum:  ReleaseAsset{Name: ChecksumAssetName, BrowserDownloadURL: "https://github.com/package.sha256", Size: 80},
	}
	destination := filepath.Join(t.TempDir(), PackageAssetName)
	if err := client.Download(context.Background(), update, destination); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != packageBody {
		t.Fatalf("downloaded %q, want %q", data, packageBody)
	}
}

func TestClientDownloadRejectsChecksumMismatch(t *testing.T) {
	t.Parallel()
	client := NewClient("v1.1.0")
	client.HTTPClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if strings.HasSuffix(request.URL.Path, ".sha256") {
			return response(http.StatusOK, strings.Repeat("0", 64)), nil
		}
		return response(http.StatusOK, "package"), nil
	})
	update := Update{
		Package:  ReleaseAsset{Name: PackageAssetName, BrowserDownloadURL: "https://github.com/package", Size: 7},
		Checksum: ReleaseAsset{Name: ChecksumAssetName, BrowserDownloadURL: "https://github.com/package.sha256", Size: 64},
	}
	err := client.Download(context.Background(), update, filepath.Join(t.TempDir(), PackageAssetName))
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("Download() error = %v, want checksum mismatch", err)
	}
}
