package security

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrChecksumMismatch indicates a file's SHA-256 digest did not match the expected value.
var ErrChecksumMismatch = errors.New("sha256 checksum mismatch")

// ComputeSHA256 returns the lowercase hex SHA-256 digest of data.
func ComputeSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// ComputeFileSHA256 streams path through SHA-256 using constant memory.
func ComputeFileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening file for sha256: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("hashing file %q: %w", path, err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// VerifyFileSHA256 checks that path matches expectedHex in constant time.
func VerifyFileSHA256(path, expectedHex string) error {
	cleanExpected := strings.ToLower(strings.TrimSpace(expectedHex))
	if cleanExpected == "" {
		return errors.New("expected sha256 checksum cannot be empty")
	}
	actual, err := ComputeFileSHA256(path)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(actual), []byte(cleanExpected)) != 1 {
		return fmt.Errorf("%w for %q: got %s, expected %s", ErrChecksumMismatch, path, actual, cleanExpected)
	}
	return nil
}

// SafeDownloader performs bounded, timeout-protected HTTP downloads with optional SHA-256 verification.
type SafeDownloader struct {
	client   *http.Client
	maxBytes int64
}

// NewSafeDownloader constructs a SafeDownloader with a tuned HTTP transport.
func NewSafeDownloader(timeout time.Duration, maxBytes int64) *SafeDownloader {
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	if maxBytes <= 0 {
		maxBytes = 512 * 1024 * 1024 // 512 MB cap
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          16,
		MaxIdleConnsPerHost:   8,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
	}
	return &SafeDownloader{
		client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		maxBytes: maxBytes,
	}
}

// DownloadToFile downloads url into a temporary sibling file, verifies expectedSHA256 if non-empty, and atomically renames to destPath.
func (d *SafeDownloader) DownloadToFile(
	ctx context.Context,
	url string,
	destPath string,
	expectedSHA256 string,
) error {
	if url == "" || destPath == "" {
		return errors.New("url and destination path are required")
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("creating download parent directory: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "anx-env-manager/2.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("downloading %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("downloading %q: unexpected HTTP status %d", url, resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(destPath), ".anx-dl-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temporary download file: %w", err)
	}
	tmpPath := tmpFile.Name()

	hasher := sha256.New()
	limited := io.LimitReader(resp.Body, d.maxBytes+1)
	written, copyErr := io.Copy(io.MultiWriter(tmpFile, hasher), limited)
	closeErr := tmpFile.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("writing downloaded payload: %w", errors.Join(copyErr, closeErr))
	}
	if written > d.maxBytes {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("download from %q exceeded maximum size %d bytes", url, d.maxBytes)
	}

	if cleanExpected := strings.ToLower(strings.TrimSpace(expectedSHA256)); cleanExpected != "" {
		actual := hex.EncodeToString(hasher.Sum(nil))
		if subtle.ConstantTimeCompare([]byte(actual), []byte(cleanExpected)) != 1 {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("%w for %q: got %s, expected %s", ErrChecksumMismatch, url, actual, cleanExpected)
		}
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("moving verified download to %q: %w", destPath, err)
	}
	return nil
}
