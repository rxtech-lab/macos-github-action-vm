package updater

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultReleaseURL  = "https://api.github.com/repos/rxtech-lab/macos-github-action-vm/releases/latest"
	PackageAssetName   = "rvmm_macOS_arm64.pkg"
	ChecksumAssetName  = PackageAssetName + ".sha256"
	maxReleaseBodySize = 2 << 20
	maxChecksumSize    = 16 << 10
	maxPackageSize     = 512 << 20
)

type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type githubRelease struct {
	TagName    string         `json:"tag_name"`
	Draft      bool           `json:"draft"`
	Prerelease bool           `json:"prerelease"`
	Assets     []ReleaseAsset `json:"assets"`
}

type Update struct {
	CurrentVersion string
	LatestVersion  string
	Package        ReleaseAsset
	Checksum       ReleaseAsset
	Available      bool
}

type ReleaseClient interface {
	Check(context.Context, string) (Update, error)
	Download(context.Context, Update, string) error
}

type Client struct {
	ReleaseURL string
	HTTPClient *http.Client
	UserAgent  string
}

func NewClient(version string) *Client {
	return &Client{
		ReleaseURL: DefaultReleaseURL,
		HTTPClient: &http.Client{
			Timeout: 2 * time.Minute,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return errors.New("too many redirects")
				}
				if req.URL.Scheme != "https" || !allowedDownloadHost(req.URL.Hostname()) {
					return fmt.Errorf("refusing redirect to %s", req.URL.String())
				}
				return nil
			},
		},
		UserAgent: "rvmm/" + version,
	}
}

func (c *Client) Check(ctx context.Context, currentVersion string) (Update, error) {
	if _, err := parseVersion(currentVersion); err != nil {
		return Update{}, fmt.Errorf("current build is not a release version: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.ReleaseURL, nil)
	if err != nil {
		return Update{}, fmt.Errorf("create release request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return Update{}, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return Update{}, fmt.Errorf("fetch latest release: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var release githubRelease
	decoder := json.NewDecoder(io.LimitReader(resp.Body, maxReleaseBodySize))
	if err := decoder.Decode(&release); err != nil {
		return Update{}, fmt.Errorf("decode latest release: %w", err)
	}
	if release.Draft || release.Prerelease {
		return Update{}, errors.New("latest release must be a published full release")
	}
	if _, err := parseVersion(release.TagName); err != nil {
		return Update{}, fmt.Errorf("latest release tag: %w", err)
	}

	comparison, err := compareVersions(release.TagName, currentVersion)
	if err != nil {
		return Update{}, fmt.Errorf("compare versions: %w", err)
	}
	result := Update{CurrentVersion: currentVersion, LatestVersion: release.TagName, Available: comparison > 0}
	if !result.Available {
		return result, nil
	}

	for _, asset := range release.Assets {
		switch asset.Name {
		case PackageAssetName:
			result.Package = asset
		case ChecksumAssetName:
			result.Checksum = asset
		}
	}
	if err := validateAsset(result.Package, PackageAssetName, maxPackageSize); err != nil {
		return Update{}, err
	}
	if err := validateAsset(result.Checksum, ChecksumAssetName, maxChecksumSize); err != nil {
		return Update{}, err
	}
	return result, nil
}

func (c *Client) Download(ctx context.Context, update Update, destination string) error {
	expectedChecksum, err := c.fetchChecksum(ctx, update.Checksum.BrowserDownloadURL)
	if err != nil {
		return err
	}

	if err := validateDownloadURL(update.Package.BrowserDownloadURL); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, update.Package.BrowserDownloadURL, nil)
	if err != nil {
		return fmt.Errorf("create package request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("download package: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download package: status %d", resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return fmt.Errorf("create download directory: %w", err)
	}
	temporary := destination + ".part"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("create package file: %w", err)
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(resp.Body, maxPackageSize+1))
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("write package: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("close package: %w", closeErr)
	}
	if written > maxPackageSize || (update.Package.Size > 0 && written != update.Package.Size) {
		_ = os.Remove(temporary)
		return fmt.Errorf("package size mismatch: downloaded %d bytes, release reports %d", written, update.Package.Size)
	}
	actualChecksum := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actualChecksum, expectedChecksum) {
		_ = os.Remove(temporary)
		return fmt.Errorf("package checksum mismatch: got %s, expected %s", actualChecksum, expectedChecksum)
	}
	if err := os.Rename(temporary, destination); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("finalize package: %w", err)
	}
	return nil
}

func (c *Client) fetchChecksum(ctx context.Context, checksumURL string) (string, error) {
	if err := validateDownloadURL(checksumURL); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		return "", fmt.Errorf("create checksum request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download checksum: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download checksum: status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(io.LimitReader(resp.Body, maxChecksumSize))
	if !scanner.Scan() {
		return "", errors.New("checksum asset is empty")
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) == 0 || len(fields[0]) != sha256.Size*2 {
		return "", errors.New("checksum asset does not contain a SHA-256 digest")
	}
	if _, err := hex.DecodeString(fields[0]); err != nil {
		return "", fmt.Errorf("invalid SHA-256 digest: %w", err)
	}
	return fields[0], nil
}

func validateAsset(asset ReleaseAsset, name string, maximumSize int64) error {
	if asset.Name != name || asset.BrowserDownloadURL == "" {
		return fmt.Errorf("release is missing required asset %s", name)
	}
	if asset.Size <= 0 || asset.Size > maximumSize {
		return fmt.Errorf("release asset %s has invalid size %d", name, asset.Size)
	}
	return validateDownloadURL(asset.BrowserDownloadURL)
}

func validateDownloadURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid release asset URL: %w", err)
	}
	if parsed.Scheme != "https" || !allowedDownloadHost(parsed.Hostname()) {
		return fmt.Errorf("refusing release asset URL %s", rawURL)
	}
	return nil
}

func allowedDownloadHost(host string) bool {
	return host == "github.com" || host == "objects.githubusercontent.com" || host == "release-assets.githubusercontent.com" || host == "github-releases.githubusercontent.com"
}
