package extension

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
)

// Admission errors. All of them are returned BEFORE the extension process is started.
var (
	// ErrNotPinned indicates the Spec carries no SHA-256 pin; unpinned binaries are never executed.
	ErrNotPinned = errors.New("extension binary is not pinned by SHA-256")
	// ErrIntegrity indicates the on-disk binary does not match its pinned SHA-256.
	ErrIntegrity = errors.New("extension binary integrity check failed: SHA-256 mismatch")
	// ErrUntrustedBinary indicates the binary or its directory fails the ownership/permission policy.
	ErrUntrustedBinary = errors.New("extension binary location is not trusted")
)

// admit validates spec against the host policy and returns the resolved binary path to execute.
// It performs no execution of the candidate binary.
func admit(spec Spec) (string, error) {
	if spec.Path == "" || !filepath.IsAbs(spec.Path) {
		return "", fmt.Errorf("%w: path %q must be absolute", ErrUntrustedBinary, spec.Path)
	}

	// Resolve symlinks so the file we hash and check is the file we execute.
	resolved, err := filepath.EvalSymlinks(spec.Path)
	if err != nil {
		return "", fmt.Errorf("%w: resolving %q: %v", ErrUntrustedBinary, spec.Path, err)
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUntrustedBinary, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%w: %q is not a regular file", ErrUntrustedBinary, resolved)
	}

	// Cheap checks first: permissions are refused before spending I/O on hashing.
	if err := checkPermissionSubset(spec.Declared, spec.Allowed); err != nil {
		return "", err
	}
	if err := checkFileTrust(resolved); err != nil {
		return "", err
	}
	if err := verifyPin(resolved, spec.SHA256); err != nil {
		return "", err
	}
	return resolved, nil
}

// checkPermissionSubset requires every declared permission to be allowed by the host.
func checkPermissionSubset(declared, allowed []string) error {
	for _, p := range declared {
		if !slices.Contains(allowed, p) {
			return fmt.Errorf("%w: declared unpermitted %q", ErrPermissionDenied, p)
		}
	}
	return nil
}

// verifyPin streams the file through SHA-256 and compares it to the pin in constant time.
func verifyPin(path, pinHex string) error {
	if pinHex == "" {
		return ErrNotPinned
	}
	want, err := hex.DecodeString(pinHex)
	if err != nil || len(want) != sha256.Size {
		return fmt.Errorf("%w: pin is not a valid SHA-256 hex digest", ErrNotPinned)
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUntrustedBinary, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hashing extension binary: %w", err)
	}
	if subtle.ConstantTimeCompare(h.Sum(nil), want) != 1 {
		return ErrIntegrity
	}
	return nil
}

// HashFile returns the lowercase hex SHA-256 of the file at path; operators use it to pin a binary.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
