package selfupdate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Planning and apply errors.
var (
	// ErrNotUpdatable indicates the binary was built locally; the repository checkout owns its updates.
	ErrNotUpdatable = errors.New("this binary was built from source and is not managed by release updates")
	// ErrAssetMissing indicates the release lacks a file required for a verified update.
	ErrAssetMissing = errors.New("release is missing a required asset")
	// ErrInsecureURL indicates an asset URL that is neither HTTPS nor loopback.
	ErrInsecureURL = errors.New("refusing to download from a non-HTTPS URL")
	// ErrGoUnavailable indicates the go-install method was requested but no Go toolchain is on PATH.
	ErrGoUnavailable = errors.New("go toolchain not found on PATH")
)

// Method is the mechanism used to obtain the new binary.
type Method string

const (
	// MethodAuto picks a method from the binary's origin.
	MethodAuto Method = "auto"
	// MethodRelease downloads, verifies and installs a published release archive.
	MethodRelease Method = "release"
	// MethodGoInstall builds the tagged module with the Go toolchain (verified by the Go checksum database).
	MethodGoInstall Method = "go-install"
	// MethodNone means no automatic update is possible for this binary.
	MethodNone Method = "none"
)

const (
	defaultMaxArchiveBytes = 200 << 20
	defaultMaxBinaryBytes  = 300 << 20
)

// Updater plans and applies binary updates. Every external effect is injectable.
type Updater struct {
	Source    Source
	Client    *http.Client
	Build     BuildInfo
	ExecPath  string
	GOOS      string
	GOARCH    string
	PublicKey string // base64 ed25519; defaults to the build-time embedded key

	// Hooks for tests and alternative environments; defaults are used when nil.
	SelfTest     func(ctx context.Context, path string) error
	LookPathGo   func() (string, error)
	RunGoInstall func(ctx context.Context, goBin, tag, gobin string) error
	Logf         func(format string, args ...any)

	MaxArchiveBytes int64
	MaxBinaryBytes  int64
}

// Request narrows what Check resolves.
type Request struct {
	// Tag pins an exact release; empty means latest. An explicit tag may downgrade.
	Tag               string
	IncludePrerelease bool
	Method            Method
}

// Plan is the resolved, side-effect-free description of an update.
type Plan struct {
	Current  string
	Target   Release
	Version  Version
	UpToDate bool
	Method   Method
	// Reason explains MethodNone.
	Reason string

	Archive   Asset
	Checksums Asset
	Signature Asset
	// SignatureRequired is true when the build embeds a release key: an unsigned release is refused.
	SignatureRequired bool

	ExecPath string
	Notes    []string
}

// Result reports a completed update.
type Result struct {
	From, To string
	Method   Method
	Backup   string
	Signed   bool
	ExecPath string
}

// New builds an Updater with production defaults.
func New(build BuildInfo, execPath string, src Source) *Updater {
	return &Updater{
		Source:   src,
		Client:   http.DefaultClient,
		Build:    build,
		ExecPath: execPath,
		GOOS:     runtime.GOOS,
		GOARCH:   runtime.GOARCH,
	}
}

func (u *Updater) logf(format string, args ...any) {
	if u.Logf != nil {
		u.Logf(format, args...)
	}
}

func (u *Updater) publicKey() string {
	if u.PublicKey != "" {
		return u.PublicKey
	}
	return releasePublicKey
}

// Check resolves what an update would do without modifying anything or downloading assets.
func (u *Updater) Check(ctx context.Context, req Request) (*Plan, error) {
	rel, err := u.release(ctx, req)
	if err != nil {
		return nil, err
	}

	target, err := ParseVersion(rel.Tag)
	if err != nil {
		return nil, fmt.Errorf("release tag %q: %w", rel.Tag, err)
	}

	plan := &Plan{Current: u.Build.Version, Target: rel, Version: target, ExecPath: u.ExecPath}

	current, curErr := ParseVersion(u.Build.Version)
	if curErr != nil {
		plan.Notes = append(plan.Notes, fmt.Sprintf("current version %q is not semantic; treating as older than any release", u.Build.Version))
	}
	cmp := target.Compare(current)
	plan.UpToDate = cmp == 0 || (cmp < 0 && req.Tag == "")
	if cmp < 0 && req.Tag != "" {
		plan.Notes = append(plan.Notes, "explicit tag is older than the running version: this is a downgrade")
	}

	u.chooseMethod(plan, req.Method)
	if plan.Method == MethodRelease && !plan.UpToDate {
		if err := u.resolveAssets(plan); err != nil {
			return plan, err
		}
	}
	return plan, nil
}

func (u *Updater) release(ctx context.Context, req Request) (Release, error) {
	if req.Tag != "" {
		return u.Source.ByTag(ctx, req.Tag)
	}
	return u.Source.Latest(ctx, req.IncludePrerelease)
}

func (u *Updater) chooseMethod(plan *Plan, requested Method) {
	switch requested {
	case MethodRelease, MethodGoInstall:
		plan.Method = requested
		return
	}

	switch u.Build.Origin {
	case OriginRelease:
		plan.Method = MethodRelease
	case OriginGoInstall:
		plan.Method = MethodGoInstall
		if _, err := u.lookGo(); err != nil {
			plan.Method = MethodRelease
			plan.Notes = append(plan.Notes, "go toolchain not found; using the release archive instead of go install")
		}
	default:
		plan.Method = MethodNone
		plan.Reason = "built from source: run `anx upgrade` to pull the repository and rebuild, " +
			"or pass --method release|go-install to replace it with a published build"
	}
}

func (u *Updater) resolveAssets(plan *Plan) error {
	name, err := ArchiveName(u.GOOS, u.GOARCH)
	if err != nil {
		return err
	}

	var ok bool
	if plan.Archive, ok = plan.Target.Asset(name); !ok {
		return fmt.Errorf("%w: %s in release %s", ErrAssetMissing, name, plan.Target.Tag)
	}
	if plan.Checksums, ok = plan.Target.Asset(ChecksumsName); !ok {
		return fmt.Errorf("%w: %s in release %s (unverifiable releases are never installed)",
			ErrAssetMissing, ChecksumsName, plan.Target.Tag)
	}

	if u.publicKey() != "" {
		plan.SignatureRequired = true
		if plan.Signature, ok = plan.Target.Asset(SignatureName); !ok {
			return fmt.Errorf("%w: %s in release %s (this build requires signed releases)",
				ErrAssetMissing, SignatureName, plan.Target.Tag)
		}
	}
	return nil
}

func (u *Updater) lookGo() (string, error) {
	if u.LookPathGo != nil {
		return u.LookPathGo()
	}
	p, err := exec.LookPath("go")
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrGoUnavailable, err)
	}
	return p, nil
}

// Apply executes plan. The running binary is replaced only after the new one is fully verified
// and has proven it can start.
func (u *Updater) Apply(ctx context.Context, plan *Plan) (*Result, error) {
	if plan.UpToDate {
		return &Result{From: plan.Current, To: plan.Current, Method: plan.Method, ExecPath: plan.ExecPath}, nil
	}
	if plan.Method == MethodNone {
		return nil, fmt.Errorf("%w: %s", ErrNotUpdatable, plan.Reason)
	}

	stageDir, err := os.MkdirTemp(filepath.Dir(plan.ExecPath), ".anx-update-*")
	if err != nil {
		return nil, explainPermission(fmt.Errorf("creating staging directory: %w", err), plan.ExecPath)
	}
	defer os.RemoveAll(stageDir)

	staged := filepath.Join(stageDir, executableName(u.GOOS))
	signed := false
	switch plan.Method {
	case MethodRelease:
		signed, err = u.stageRelease(ctx, plan, stageDir, staged)
	case MethodGoInstall:
		err = u.stageGoInstall(ctx, plan, stageDir, staged)
	default:
		err = fmt.Errorf("unknown update method %q", plan.Method)
	}
	if err != nil {
		return nil, err
	}

	if err := os.Chmod(staged, 0o755); err != nil {
		return nil, fmt.Errorf("marking staged binary executable: %w", err)
	}
	selfTest := u.selfTest()
	u.logf("self-testing staged binary")
	if err := selfTest(ctx, staged); err != nil {
		return nil, fmt.Errorf("staged binary rejected: %w", err)
	}

	u.logf("replacing %s", plan.ExecPath)
	backup, err := replaceBinary(ctx, staged, plan.ExecPath, selfTest)
	if err != nil {
		return nil, err
	}
	return &Result{
		From: plan.Current, To: plan.Target.Tag, Method: plan.Method,
		Backup: backup, Signed: signed, ExecPath: plan.ExecPath,
	}, nil
}

func (u *Updater) selfTest() func(context.Context, string) error {
	if u.SelfTest != nil {
		return u.SelfTest
	}
	return selfTestBinary
}

func executableName(goos string) string {
	if goos == "windows" {
		return BinaryName + ".exe"
	}
	return BinaryName
}

// stageRelease downloads, verifies and extracts the release archive. It reports whether the
// checksum list was authenticated by a signature.
func (u *Updater) stageRelease(ctx context.Context, plan *Plan, stageDir, staged string) (signed bool, err error) {
	u.logf("downloading %s", plan.Checksums.Name)
	sumsData, err := u.fetchBytes(ctx, plan.Checksums.URL, maxChecksumFileBytes)
	if err != nil {
		return false, err
	}

	if plan.SignatureRequired {
		u.logf("verifying release signature")
		sig, err := u.fetchBytes(ctx, plan.Signature.URL, 4096)
		if err != nil {
			return false, err
		}
		if err := VerifySignature(sumsData, sig, u.publicKey()); err != nil {
			return false, err
		}
		signed = true
	}

	sums, err := ParseChecksums(bytes.NewReader(sumsData))
	if err != nil {
		return false, err
	}

	archive := filepath.Join(stageDir, plan.Archive.Name)
	u.logf("downloading %s", plan.Archive.Name)
	digest, err := u.fetchToFile(ctx, plan.Archive.URL, archive, u.maxArchive())
	if err != nil {
		return false, err
	}
	if err := verifyDigest(sums, plan.Archive.Name, digest); err != nil {
		return false, err
	}

	u.logf("extracting %s", BinaryName)
	if err := extractBinary(archive, executableName(u.GOOS), staged, u.maxBinary()); err != nil {
		return false, err
	}
	return signed, nil
}

func (u *Updater) maxArchive() int64 {
	if u.MaxArchiveBytes > 0 {
		return u.MaxArchiveBytes
	}
	return defaultMaxArchiveBytes
}

func (u *Updater) maxBinary() int64 {
	if u.MaxBinaryBytes > 0 {
		return u.MaxBinaryBytes
	}
	return defaultMaxBinaryBytes
}

// stageGoInstall builds the tagged module into stageDir. The Go checksum database authenticates
// the module source, so no extra signature is needed for this path.
func (u *Updater) stageGoInstall(ctx context.Context, plan *Plan, stageDir, staged string) error {
	goBin, err := u.lookGo()
	if err != nil {
		return err
	}

	u.logf("building %s@%s with %s", CmdPackage, plan.Target.Tag, goBin)
	run := u.RunGoInstall
	if run == nil {
		run = runGoInstall
	}
	if err := run(ctx, goBin, plan.Target.Tag, stageDir); err != nil {
		return err
	}

	built := filepath.Join(stageDir, executableName(u.GOOS))
	if built != staged {
		return os.Rename(built, staged)
	}
	if _, err := os.Stat(staged); err != nil {
		return fmt.Errorf("go install produced no %s binary: %w", BinaryName, err)
	}
	return nil
}

func runGoInstall(ctx context.Context, goBin, tag, gobin string) error {
	cmd := exec.CommandContext(ctx, goBin, "install", "-trimpath", CmdPackage+"@"+tag)
	cmd.Dir = gobin // keep the invoking directory's go.mod out of module resolution
	cmd.Env = append(os.Environ(), "GOBIN="+gobin, "GOFLAGS=-mod=mod")

	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go install %s@%s failed: %w\n%s", CmdPackage, tag, err, strings.TrimSpace(out.String()))
	}
	return nil
}

// allowedURL permits HTTPS, and plain HTTP only to loopback (local mirrors and tests).
func allowedURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("%w: %q", ErrInsecureURL, raw)
	}
	if u.Scheme == "https" {
		return nil
	}
	if u.Scheme == "http" {
		host := u.Hostname()
		if ip := net.ParseIP(host); host == "localhost" || (ip != nil && ip.IsLoopback()) {
			return nil
		}
	}
	return fmt.Errorf("%w: %q", ErrInsecureURL, raw)
}

func (u *Updater) get(ctx context.Context, rawURL string) (*http.Response, error) {
	if err := allowedURL(rawURL); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "anx-self-update")

	client := u.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", rawURL, err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("downloading %s: %s", rawURL, resp.Status)
	}
	return resp, nil
}

// fetchBytes downloads a small asset fully into memory, failing if it exceeds max bytes.
func (u *Updater) fetchBytes(ctx context.Context, rawURL string, max int64) ([]byte, error) {
	resp, err := u.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", rawURL, err)
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("%s exceeds the %d byte limit", rawURL, max)
	}
	return data, nil
}

// fetchToFile streams a download to dst while hashing it, so the archive is never held in memory
// and its digest is known before anything is extracted.
func (u *Updater) fetchToFile(ctx context.Context, rawURL, dst string, max int64) ([]byte, error) {
	resp, err := u.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, err
	}

	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, max+1))
	closeErr := f.Close()
	switch {
	case copyErr != nil:
		return nil, fmt.Errorf("downloading %s: %w", rawURL, copyErr)
	case n > max:
		return nil, fmt.Errorf("%s exceeds the %d byte limit", rawURL, max)
	case closeErr != nil:
		return nil, closeErr
	}
	return h.Sum(nil), nil
}
