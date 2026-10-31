package security

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/manifests"
)

// ErrTamperDetected indicates an asset was modified, added, or deleted after the seal was established.
var ErrTamperDetected = errors.New("security seal violation: manifest or dotfile asset has been tampered with or modified on disk")

// ErrUntrustedOrigin indicates the repository or manifest directory has insecure permissions.
var ErrUntrustedOrigin = errors.New("security seal violation: repository origin directory has insecure permissions or untrusted ownership")

var hasherPool = sync.Pool{
	New: func() any {
		return sha256.New()
	},
}

// FileDigest records the relative path and cryptographic hash of an asset.
type FileDigest struct {
	RelPath string `json:"rel_path"`
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size"`
}

// OriginSeal represents an immutable cryptographic fingerprint and provenance record
// of all active manifests and dotfiles.
type OriginSeal struct {
	RepoRoot            string                `json:"repo_root"`
	IsEmbedded          bool                  `json:"is_embedded"`
	UserDropInDir       string                `json:"user_drop_in_dir,omitempty"`
	TargetHome          string                `json:"target_home"`
	ManifestFingerprint string                `json:"manifest_fingerprint"`
	DotfilesFingerprint string                `json:"dotfiles_fingerprint"`
	CompositeDigest     string                `json:"composite_digest"`
	AssetCount          int                   `json:"asset_count"`
	SealedAt            time.Time             `json:"sealed_at"`
	InsecureOrigin      bool                  `json:"insecure_origin"`
	InsecureReason      string                `json:"insecure_reason,omitempty"`
	OriginTrust         string                `json:"origin_trust,omitempty"`
	OriginTrustWarning  string                `json:"origin_trust_warning,omitempty"`
	ManifestDigests     map[string]FileDigest `json:"manifest_digests,omitempty"`
	DotfileDigests      map[string]FileDigest `json:"dotfile_digests,omitempty"`
	mu                  sync.RWMutex
	trust               DirectoryTrustVerifier
}

// SealOptions configures OriginSeal generation.
type SealOptions struct {
	RepoRoot      string
	UserDropInDir string
	TargetHome    string
	SkipDotfiles  bool
	Trust         DirectoryTrustVerifier
}

// SealOption configures a SealOptions value.
type SealOption func(*SealOptions)

// WithSealRepoRoot sets the root repository path.
func WithSealRepoRoot(dir string) SealOption {
	return func(o *SealOptions) {
		o.RepoRoot = strings.TrimSpace(dir)
	}
}

// WithSealUserDropIn sets the user drop-in manifests directory.
func WithSealUserDropIn(dir string) SealOption {
	return func(o *SealOptions) {
		o.UserDropInDir = strings.TrimSpace(dir)
	}
}

// WithSealTargetHome sets the target home destination.
func WithSealTargetHome(dir string) SealOption {
	return func(o *SealOptions) {
		o.TargetHome = strings.TrimSpace(dir)
	}
}

// WithSealTrustVerifier overrides the OS-specific directory trust verifier (used by tests and policy hooks).
func WithSealTrustVerifier(v DirectoryTrustVerifier) SealOption {
	return func(o *SealOptions) {
		o.Trust = v
	}
}

// WithSealSkipDotfiles skips scanning dotfile source templates in files/.
func WithSealSkipDotfiles(skip bool) SealOption {
	return func(o *SealOptions) {
		o.SkipDotfiles = skip
	}
}

// CreateOriginSeal builds a deterministic, cryptographic integrity seal over all manifest
// files and dotfile templates. The operation runs in single-digit milliseconds.
func CreateOriginSeal(opts ...SealOption) (*OriginSeal, error) {
	cfg := &SealOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	if cfg.TargetHome == "" {
		if home, err := os.UserHomeDir(); err == nil {
			cfg.TargetHome = home
		}
	}

	if cfg.Trust == nil {
		cfg.Trust = DefaultTrustVerifier()
	}

	seal := &OriginSeal{
		trust:           cfg.Trust,
		RepoRoot:        cfg.RepoRoot,
		UserDropInDir:   cfg.UserDropInDir,
		TargetHome:      cfg.TargetHome,
		SealedAt:        time.Now().UTC(),
		ManifestDigests: make(map[string]FileDigest, 64),
		DotfileDigests:  make(map[string]FileDigest, 64),
	}

	// 1. Check repository trust & ownership
	if cfg.RepoRoot != "" {
		report := cfg.Trust.VerifyDirectory(cfg.RepoRoot)
		seal.OriginTrust = report.Level.String()
		seal.InsecureOrigin = report.Level.Blocks()
		switch {
		case seal.InsecureOrigin:
			seal.InsecureReason = report.Reason()
		case report.Level == TrustDegraded:
			seal.OriginTrustWarning = report.Reason()
		}
	}

	// 2. Hash embedded manifests (immutable baseline)
	if err := hashEmbeddedManifests(seal.ManifestDigests); err != nil {
		return nil, fmt.Errorf("hashing embedded manifests: %w", err)
	}

	// 3. Hash repository manifests (if present)
	if cfg.RepoRoot != "" {
		repoManDir := filepath.Join(cfg.RepoRoot, "manifests")
		if info, err := os.Stat(repoManDir); err == nil && info.IsDir() {
			seal.IsEmbedded = false
			if err := hashOSDirectory(repoManDir, "repo:manifests", seal.ManifestDigests, []string{".yaml", ".yml"}); err != nil {
				return nil, fmt.Errorf("hashing repo manifests: %w", err)
			}
		} else {
			seal.IsEmbedded = true
		}
	} else {
		seal.IsEmbedded = true
	}

	// 4. Hash user drop-in manifests (if present)
	if cfg.UserDropInDir != "" {
		if info, err := os.Stat(cfg.UserDropInDir); err == nil && info.IsDir() {
			if err := hashOSDirectory(cfg.UserDropInDir, "user:manifests.d", seal.ManifestDigests, []string{".yaml", ".yml"}); err != nil {
				return nil, fmt.Errorf("hashing user drop-in manifests: %w", err)
			}
		}
	}

	// 5. Hash dotfile source templates (if not skipped and repo has files/)
	if !cfg.SkipDotfiles && cfg.RepoRoot != "" {
		filesDir := filepath.Join(cfg.RepoRoot, "files")
		if info, err := os.Stat(filesDir); err == nil && info.IsDir() {
			if err := hashOSDirectory(filesDir, "files", seal.DotfileDigests, nil); err != nil {
				return nil, fmt.Errorf("hashing dotfile sources: %w", err)
			}
		}
	}

	// 6. Compute composite fingerprints
	seal.ManifestFingerprint = computeTreeFingerprint(seal.ManifestDigests)
	seal.DotfilesFingerprint = computeTreeFingerprint(seal.DotfileDigests)
	seal.CompositeDigest = computeComposite(seal.ManifestFingerprint, seal.DotfilesFingerprint)
	seal.AssetCount = len(seal.ManifestDigests) + len(seal.DotfileDigests)

	return seal, nil
}

// verifier returns the configured trust verifier, defaulting to the OS implementation.
func (s *OriginSeal) verifier() DirectoryTrustVerifier {
	if s.trust == nil {
		return DefaultTrustVerifier()
	}
	return s.trust
}

// ShortDigest returns an abbreviated 12-hex-character digest for clean UI badges.
func (s *OriginSeal) ShortDigest() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.CompositeDigest) > 12 {
		return s.CompositeDigest[:12]
	}
	return s.CompositeDigest
}

// ShortManifestDigest returns an abbreviated 12-character manifest digest.
func (s *OriginSeal) ShortManifestDigest() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.ManifestFingerprint) > 12 {
		return s.ManifestFingerprint[:12]
	}
	return s.ManifestFingerprint
}

// ShortDotfilesDigest returns an abbreviated 12-character dotfiles digest.
func (s *OriginSeal) ShortDotfilesDigest() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.DotfilesFingerprint) > 12 {
		return s.DotfilesFingerprint[:12]
	}
	return s.DotfilesFingerprint
}

// Verify re-inspects active manifest files and dotfile templates on disk and guarantees
// that no content, additions, deletions, or switches have occurred since sealing.
func (s *OriginSeal) Verify(_ context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Recheck repo trust if repo is active
	if s.RepoRoot != "" {
		if report := s.verifier().VerifyDirectory(s.RepoRoot); report.Level.Blocks() {
			return domain.NewExitError(
				domain.ExitSecurityViolation,
				fmt.Errorf("%w: %s (%s)", ErrUntrustedOrigin, report.Reason(), report.Level),
			)
		}
	}

	freshManifests := make(map[string]FileDigest, len(s.ManifestDigests))
	freshDotfiles := make(map[string]FileDigest, len(s.DotfileDigests))

	// 1. Hash embedded manifests
	if err := hashEmbeddedManifests(freshManifests); err != nil {
		return fmt.Errorf("re-hashing embedded manifests: %w", err)
	}

	// 2. Hash repo manifests
	if s.RepoRoot != "" {
		repoManDir := filepath.Join(s.RepoRoot, "manifests")
		if info, err := os.Stat(repoManDir); err == nil && info.IsDir() {
			if err := hashOSDirectory(repoManDir, "repo:manifests", freshManifests, []string{".yaml", ".yml"}); err != nil {
				return domain.NewExitError(
					domain.ExitSecurityViolation,
					fmt.Errorf("%w: failed reading repo manifests: %v", ErrTamperDetected, err),
				)
			}
		}
	}

	// 3. Hash user drop-in manifests
	if s.UserDropInDir != "" {
		if info, err := os.Stat(s.UserDropInDir); err == nil && info.IsDir() {
			if err := hashOSDirectory(s.UserDropInDir, "user:manifests.d", freshManifests, []string{".yaml", ".yml"}); err != nil {
				return domain.NewExitError(
					domain.ExitSecurityViolation,
					fmt.Errorf("%w: failed reading user drop-in manifests: %v", ErrTamperDetected, err),
				)
			}
		}
	}

	// 4. Hash dotfile templates
	if len(s.DotfileDigests) > 0 && s.RepoRoot != "" {
		filesDir := filepath.Join(s.RepoRoot, "files")
		if info, err := os.Stat(filesDir); err == nil && info.IsDir() {
			if err := hashOSDirectory(filesDir, "files", freshDotfiles, nil); err != nil {
				return domain.NewExitError(
					domain.ExitSecurityViolation,
					fmt.Errorf("%w: failed reading dotfile assets: %v", ErrTamperDetected, err),
				)
			}
		}
	}

	// 5. Compare manifest fingerprint
	freshManifestFP := computeTreeFingerprint(freshManifests)
	if subtle.ConstantTimeCompare([]byte(freshManifestFP), []byte(s.ManifestFingerprint)) != 1 {
		diffKey := findFirstDiscrepancy(s.ManifestDigests, freshManifests)
		return domain.NewExitError(
			domain.ExitSecurityViolation,
			fmt.Errorf("%w: manifest asset %q was modified on disk after sealing", ErrTamperDetected, diffKey),
		)
	}

	// 6. Compare dotfiles fingerprint
	freshDotfilesFP := computeTreeFingerprint(freshDotfiles)
	if subtle.ConstantTimeCompare([]byte(freshDotfilesFP), []byte(s.DotfilesFingerprint)) != 1 {
		diffKey := findFirstDiscrepancy(s.DotfileDigests, freshDotfiles)
		return domain.NewExitError(
			domain.ExitSecurityViolation,
			fmt.Errorf("%w: dotfile asset %q was modified on disk after sealing", ErrTamperDetected, diffKey),
		)
	}

	return nil
}

// ProvenanceOrigin returns a concise label of where the active declarations originate.
func (s *OriginSeal) ProvenanceOrigin() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.RepoRoot == "" || s.IsEmbedded {
		return "Embedded Catalog (Immutable Binary)"
	}
	return s.RepoRoot
}

func hashEmbeddedManifests(out map[string]FileDigest) error {
	return fs.WalkDir(manifests.EmbeddedFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		data, rErr := fs.ReadFile(manifests.EmbeddedFS, path)
		if rErr != nil {
			return rErr
		}

		hasher := hasherPool.Get().(hash.Hash)
		hasher.Reset()
		hasher.Write(data)
		digestHex := hex.EncodeToString(hasher.Sum(nil))
		hasherPool.Put(hasher)

		key := "embedded:" + filepath.ToSlash(path)
		out[key] = FileDigest{
			RelPath: key,
			SHA256:  digestHex,
			Size:    int64(len(data)),
		}
		return nil
	})
}

func hashOSDirectory(root, prefix string, out map[string]FileDigest, allowedExts []string) error {
	buf := make([]byte, 32*1024)

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip hidden git or temporary directories
			if info.Name() == ".git" || info.Name() == ".cache" {
				return filepath.SkipDir
			}
			return nil
		}

		name := info.Name()
		// Skip temporary, socket, log, and lock runtime files
		if strings.HasSuffix(name, ".sock") || strings.Contains(name, ".sock.") ||
			strings.HasSuffix(name, ".log") || strings.HasSuffix(name, ".lock") ||
			strings.HasSuffix(name, ".tmp") {
			return nil
		}

		// Only hash regular files (skip sockets, pipes, devices, and dangling symlinks)
		if !info.Mode().IsRegular() {
			if info.Mode()&os.ModeSymlink != 0 {
				target, sErr := filepath.EvalSymlinks(path)
				if sErr != nil {
					return nil // Skip dangling symlinks
				}
				targetInfo, sErr := os.Stat(target)
				if sErr != nil || !targetInfo.Mode().IsRegular() {
					return nil
				}
			} else {
				return nil
			}
		}

		if len(allowedExts) > 0 {
			matched := false
			for _, ext := range allowedExts {
				if strings.HasSuffix(path, ext) {
					matched = true
					break
				}
			}
			if !matched {
				return nil
			}
		}

		rel, rErr := filepath.Rel(root, path)
		if rErr != nil {
			rel = filepath.Base(path)
		}
		key := prefix + "/" + filepath.ToSlash(rel)

		file, oErr := os.Open(path)
		if oErr != nil {
			return oErr
		}
		defer file.Close()

		hasher := hasherPool.Get().(hash.Hash)
		hasher.Reset()
		_, cErr := io.CopyBuffer(hasher, file, buf)
		if cErr != nil {
			hasherPool.Put(hasher)
			return cErr
		}
		digestHex := hex.EncodeToString(hasher.Sum(nil))
		hasherPool.Put(hasher)

		out[key] = FileDigest{
			RelPath: key,
			SHA256:  digestHex,
			Size:    info.Size(),
		}
		return nil
	})
}

func computeTreeFingerprint(digests map[string]FileDigest) string {
	if len(digests) == 0 {
		return "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // sha256("")
	}

	keys := make([]string, 0, len(digests))
	for k := range digests {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	hasher := hasherPool.Get().(hash.Hash)
	hasher.Reset()
	defer hasherPool.Put(hasher)

	for _, k := range keys {
		d := digests[k]
		fmt.Fprintf(hasher, "%s:%s:%d\n", d.RelPath, d.SHA256, d.Size)
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

func computeComposite(manifestFP, dotfilesFP string) string {
	hasher := hasherPool.Get().(hash.Hash)
	hasher.Reset()
	defer hasherPool.Put(hasher)

	fmt.Fprintf(hasher, "manifests=%s\ndotfiles=%s\n", manifestFP, dotfilesFP)
	return hex.EncodeToString(hasher.Sum(nil))
}

func findFirstDiscrepancy(orig, fresh map[string]FileDigest) string {
	for k, v := range orig {
		f, ok := fresh[k]
		if !ok {
			return k + " (deleted)"
		}
		if f.SHA256 != v.SHA256 {
			return k + " (content modified)"
		}
	}
	for k := range fresh {
		if _, ok := orig[k]; !ok {
			return k + " (newly injected)"
		}
	}
	return "unknown asset"
}
