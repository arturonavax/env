package selfupdate

import (
	"bufio"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Verification errors. Every one of them aborts the update before the running binary is touched.
var (
	// ErrChecksumMismatch indicates the downloaded archive does not match SHA256SUMS.
	ErrChecksumMismatch = errors.New("archive checksum mismatch")
	// ErrChecksumMissing indicates SHA256SUMS has no entry for the archive.
	ErrChecksumMissing = errors.New("no checksum listed for archive")
	// ErrBadSignature indicates SHA256SUMS is not signed by the embedded release key.
	ErrBadSignature = errors.New("SHA256SUMS signature verification failed")
	// ErrBadChecksumFile indicates malformed SHA256SUMS content.
	ErrBadChecksumFile = errors.New("malformed SHA256SUMS")
)

const maxChecksumFileBytes = 1 << 20

// ParseChecksums reads "sha256sum" output: "<64 hex>  <name>" or "<64 hex> *<name>".
func ParseChecksums(r io.Reader) (map[string]string, error) {
	sums := make(map[string]string)

	sc := bufio.NewScanner(io.LimitReader(r, maxChecksumFileBytes))
	for line := 1; sc.Scan(); line++ {
		text := strings.TrimSpace(sc.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}

		digest, name, ok := strings.Cut(text, " ")
		name = strings.TrimPrefix(strings.TrimSpace(name), "*")
		raw, err := hex.DecodeString(digest)
		if !ok || err != nil || len(raw) != sha256.Size || name == "" {
			return nil, fmt.Errorf("%w: line %d", ErrBadChecksumFile, line)
		}
		sums[name] = strings.ToLower(digest)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("reading checksums: %w", err)
	}
	return sums, nil
}

// VerifySignature checks that sig (base64) is a valid ed25519 signature of data under pubKeyB64.
func VerifySignature(data, sig []byte, pubKeyB64 string) error {
	pub, err := base64.StdEncoding.DecodeString(strings.TrimSpace(pubKeyB64))
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("%w: embedded release key is not a valid ed25519 public key", ErrBadSignature)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sig)))
	if err != nil || len(raw) != ed25519.SignatureSize {
		return fmt.Errorf("%w: signature is not a valid base64 ed25519 signature", ErrBadSignature)
	}
	if !ed25519.Verify(pub, data, raw) {
		return ErrBadSignature
	}
	return nil
}

// verifyDigest compares a computed SHA-256 against the hex digest listed for name, in constant time.
func verifyDigest(sums map[string]string, name string, got []byte) error {
	want, ok := sums[name]
	if !ok {
		return fmt.Errorf("%w: %s", ErrChecksumMissing, name)
	}
	wantRaw, err := hex.DecodeString(want)
	if err != nil || subtle.ConstantTimeCompare(got, wantRaw) != 1 {
		return fmt.Errorf("%w: %s", ErrChecksumMismatch, name)
	}
	return nil
}
