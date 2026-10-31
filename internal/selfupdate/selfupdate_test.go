package selfupdate_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/arturonavax/env/internal/selfupdate"
)

func TestVersionParsingAndComparison(t *testing.T) {
	cases := []struct {
		input   string
		wantStr string
		isPre   bool
		wantErr bool
	}{
		{"1.0.0", "v1.0.0", false, false},
		{"v1.2.3", "v1.2.3", false, false},
		{"v2.0.0-rc.1", "v2.0.0-rc.1", true, false},
		{"v1.2", "v1.2.0", false, false},
		{"v1.0.0+meta", "v1.0.0", false, false},
		{"invalid", "", false, true},
		{"1.2.3.4", "", false, true},
		{"v1.00.0", "", false, true}, // leading zero disallowed in semver
	}

	for _, tc := range cases {
		v, err := selfupdate.ParseVersion(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseVersion(%q) expected error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseVersion(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if v.String() != tc.wantStr {
			t.Errorf("ParseVersion(%q).String() = %q, want %q", tc.input, v.String(), tc.wantStr)
		}
		if v.IsPrerelease() != tc.isPre {
			t.Errorf("ParseVersion(%q).IsPrerelease() = %v, want %v", tc.input, v.IsPrerelease(), tc.isPre)
		}
	}

	// Comparison tests
	pairs := []struct {
		a, b string
		want int // -1, 0, 1
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.1.0", "1.0.9", 1},
		{"2.0.0", "1.9.9", 1},
		{"1.0.0-alpha", "1.0.0", -1}, // prerelease is lower than release
		{"1.0.0-alpha.1", "1.0.0-alpha.2", -1},
		{"1.0.0-alpha", "1.0.0-beta", -1},
	}

	for _, p := range pairs {
		va, errA := selfupdate.ParseVersion(p.a)
		vb, errB := selfupdate.ParseVersion(p.b)
		if errA != nil || errB != nil {
			t.Fatalf("failed parsing pair %s, %s", p.a, p.b)
		}
		if got := va.Compare(vb); got != p.want {
			t.Errorf("Compare(%s, %s) = %d, want %d", p.a, p.b, got, p.want)
		}
	}
}

func TestBuildInfoResolution(t *testing.T) {
	// 1. Explicit release origin
	info := selfupdate.Resolve("v1.5.0", "abcdef1", "2026-10-31")
	if info.Version == "" {
		t.Fatal("empty version in resolved buildinfo")
	}

	// 2. Fallbacks for empty fields
	emptyInfo := selfupdate.Resolve("", "", "")
	if emptyInfo.Version != "dev" || emptyInfo.Commit != "dev" {
		t.Errorf("unexpected defaults: %+v", emptyInfo)
	}
}

func TestVerifySignatureAndChecksums(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pubB64 := base64.StdEncoding.EncodeToString(pub)

	data := []byte("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  anx_linux_amd64.tar.gz\n")
	sig := ed25519.Sign(priv, data)
	sigB64 := []byte(base64.StdEncoding.EncodeToString(sig))

	// Good signature
	if err := selfupdate.VerifySignature(data, sigB64, pubB64); err != nil {
		t.Fatalf("expected valid signature, got: %v", err)
	}

	// Bad signature
	badSigB64 := []byte(base64.StdEncoding.EncodeToString([]byte("invalid-sig-bytes-length-must-be-64-bytes-long-padding-padding!")))
	if err := selfupdate.VerifySignature(data, badSigB64, pubB64); err == nil {
		t.Fatal("expected failure on bad signature, got nil")
	}

	// Tampered data
	if err := selfupdate.VerifySignature([]byte("tampered"), sigB64, pubB64); err == nil {
		t.Fatal("expected failure on tampered data, got nil")
	}

	// ParseChecksums
	sums, err := selfupdate.ParseChecksums(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ParseChecksums failed: %v", err)
	}
	if sums["anx_linux_amd64.tar.gz"] != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatalf("unexpected parsed checksum map: %+v", sums)
	}
}

func createTarGz(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		hdr := &tar.Header{
			Name:     name,
			Mode:     0o755,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatal(err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func createZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestUpdaterEndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "anx")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	currentContent := []byte("#!/bin/sh\necho '1.0.0'\n")
	if err := os.WriteFile(binPath, currentContent, 0o755); err != nil {
		t.Fatal(err)
	}

	nextBinary := []byte("#!/bin/sh\necho '2.0.0'\n")
	archiveName, err := selfupdate.ArchiveName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skip("skipping on platform without official archive naming:", runtime.GOOS, runtime.GOARCH)
	}

	binFileName := "anx"
	if runtime.GOOS == "windows" {
		binFileName += ".exe"
	}

	var archiveBytes []byte
	if strings.HasSuffix(archiveName, ".zip") {
		archiveBytes = createZip(t, map[string][]byte{binFileName: nextBinary})
	} else {
		archiveBytes = createTarGz(t, map[string][]byte{binFileName: nextBinary})
	}

	archiveHash := sha256.Sum256(archiveBytes)
	archiveHex := hex.EncodeToString(archiveHash[:])

	checksumsContent := fmt.Sprintf("%s  %s\n", archiveHex, archiveName)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pubB64 := base64.StdEncoding.EncodeToString(pub)
	sigBytes := ed25519.Sign(priv, []byte(checksumsContent))
	sigB64 := base64.StdEncoding.EncodeToString(sigBytes)

	// Mock HTTP Server serving releases
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/archive":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(archiveBytes)
		case "/checksums":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(checksumsContent))
		case "/sig":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(sigB64))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	fakeSrc := &fakeSource{
		rel: selfupdate.Release{
			Tag: "v2.0.0",
			Assets: []selfupdate.Asset{
				{Name: archiveName, URL: server.URL + "/archive", Size: int64(len(archiveBytes))},
				{Name: selfupdate.ChecksumsName, URL: server.URL + "/checksums", Size: int64(len(checksumsContent))},
				{Name: selfupdate.SignatureName, URL: server.URL + "/sig", Size: int64(len(sigB64))},
			},
		},
	}

	build := selfupdate.BuildInfo{
		Version: "v1.0.0",
		Commit:  "abcdef0",
		Date:    "2026-10-30",
		Origin:  selfupdate.OriginRelease,
	}

	updater := selfupdate.New(build, binPath, fakeSrc)
	updater.PublicKey = pubB64
	updater.SelfTest = func(ctx context.Context, p string) error {
		// Mock successful self-test
		return nil
	}

	// 1. Check
	plan, err := updater.Check(context.Background(), selfupdate.Request{})
	if err != nil {
		t.Fatalf("updater.Check failed: %v", err)
	}
	if plan.UpToDate {
		t.Fatal("expected plan not up to date, got UpToDate=true")
	}
	if plan.Method != selfupdate.MethodRelease {
		t.Fatalf("expected MethodRelease, got %s", plan.Method)
	}

	// 2. Apply
	res, err := updater.Apply(context.Background(), plan)
	if err != nil {
		t.Fatalf("updater.Apply failed: %v", err)
	}
	if res.To != "v2.0.0" {
		t.Fatalf("expected updated version v2.0.0, got %s", res.To)
	}
	if !res.Signed {
		t.Fatal("expected release update to be marked signed")
	}

	// Check installed binary content
	installedData, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(installedData, nextBinary) {
		t.Fatalf("installed content does not match new binary: %q", installedData)
	}

	// Check backup file was created
	backupPath := binPath + ".old"
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("expected backup file %s to exist: %v", backupPath, err)
	}
	if !bytes.Equal(backupData, currentContent) {
		t.Fatalf("backup file content mismatch: expected %q, got %q", currentContent, backupData)
	}
}

func TestUpdaterRollbackOnSelfTestFailure(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "anx")
	currentContent := []byte("original working binary")
	if err := os.WriteFile(binPath, currentContent, 0o755); err != nil {
		t.Fatal(err)
	}

	archiveName, err := selfupdate.ArchiveName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skip("skipping platform:", runtime.GOOS)
	}

	binFileName := "anx"
	if runtime.GOOS == "windows" {
		binFileName += ".exe"
	}
	archiveBytes := createTarGz(t, map[string][]byte{binFileName: []byte("defective binary")})
	archiveHash := sha256.Sum256(archiveBytes)
	checksumsContent := fmt.Sprintf("%s  %s\n", hex.EncodeToString(archiveHash[:]), archiveName)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/archive":
			_, _ = w.Write(archiveBytes)
		case "/checksums":
			_, _ = w.Write([]byte(checksumsContent))
		}
	}))
	defer server.Close()

	fakeSrc := &fakeSource{
		rel: selfupdate.Release{
			Tag: "v2.0.0",
			Assets: []selfupdate.Asset{
				{Name: archiveName, URL: server.URL + "/archive"},
				{Name: selfupdate.ChecksumsName, URL: server.URL + "/checksums"},
			},
		},
	}

	updater := selfupdate.New(selfupdate.BuildInfo{
		Version: "v1.0.0",
		Origin:  selfupdate.OriginRelease,
	}, binPath, fakeSrc)

	// Inject a self-test failure
	updater.SelfTest = func(ctx context.Context, p string) error {
		if strings.Contains(p, "anx") {
			return errors.New("binary segfaulted during test")
		}
		return nil
	}

	plan, err := updater.Check(context.Background(), selfupdate.Request{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = updater.Apply(context.Background(), plan)
	if err == nil {
		t.Fatal("expected Apply to fail when self-test fails, got nil")
	}

	// Verify rollback occurred
	restored, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restored, currentContent) {
		t.Fatalf("rollback did not restore original binary: %q", restored)
	}
}

type fakeSource struct {
	rel selfupdate.Release
	err error
}

func (f *fakeSource) Latest(ctx context.Context, includePrerelease bool) (selfupdate.Release, error) {
	if f.err != nil {
		return selfupdate.Release{}, f.err
	}
	return f.rel, nil
}

func (f *fakeSource) ByTag(ctx context.Context, tag string) (selfupdate.Release, error) {
	if f.err != nil {
		return selfupdate.Release{}, f.err
	}
	return f.rel, nil
}
