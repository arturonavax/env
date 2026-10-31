package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

// ErrBinaryNotFound indicates the archive does not contain the expected executable.
var ErrBinaryNotFound = errors.New("archive does not contain the anx executable")

// ErrUnsafeArchive indicates an archive entry that could escape the extraction target or exceed limits.
var ErrUnsafeArchive = errors.New("unsafe archive")

// extractBinary copies the single executable called want out of archivePath into dst.
//
// Safety properties: only regular files are read; the destination path is chosen by the caller,
// never derived from archive content (so no path traversal or symlink tricks are possible);
// entries with absolute paths or ".." components are rejected outright; output is capped at maxBytes
// to defeat decompression bombs.
func extractBinary(archivePath, want, dst string, maxBytes int64) error {
	switch {
	case strings.HasSuffix(archivePath, ".zip"):
		return extractFromZip(archivePath, want, dst, maxBytes)
	case strings.HasSuffix(archivePath, ".tar.gz"), strings.HasSuffix(archivePath, ".tgz"):
		return extractFromTarGz(archivePath, want, dst, maxBytes)
	}
	return fmt.Errorf("%w: unrecognized archive type %q", ErrUnsafeArchive, archivePath)
}

// checkEntryName rejects names that are absolute or climb out of the archive root.
func checkEntryName(name string) error {
	clean := path.Clean(strings.ReplaceAll(name, `\`, "/"))
	if path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(name, "\x00") {
		return fmt.Errorf("%w: entry %q escapes the archive root", ErrUnsafeArchive, name)
	}
	return nil
}

func isWanted(name, want string) bool {
	return path.Base(strings.ReplaceAll(name, `\`, "/")) == want
}

func extractFromTarGz(archivePath, want, dst string, maxBytes int64) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnsafeArchive, err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	found := false
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("%w: reading tar: %v", ErrUnsafeArchive, err)
		}
		if err := checkEntryName(hdr.Name); err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg || !isWanted(hdr.Name, want) {
			continue
		}
		if found {
			return fmt.Errorf("%w: duplicate %q entries", ErrUnsafeArchive, want)
		}
		if hdr.Size > maxBytes {
			return fmt.Errorf("%w: %q is %d bytes, limit %d", ErrUnsafeArchive, hdr.Name, hdr.Size, maxBytes)
		}
		if err := writeLimited(dst, tr, maxBytes); err != nil {
			return err
		}
		found = true
	}

	if !found {
		return ErrBinaryNotFound
	}
	return nil
}

func extractFromZip(archivePath, want, dst string, maxBytes int64) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnsafeArchive, err)
	}
	defer zr.Close()

	found := false
	for _, zf := range zr.File {
		if err := checkEntryName(zf.Name); err != nil {
			return err
		}
		if !zf.Mode().IsRegular() || !isWanted(zf.Name, want) {
			continue
		}
		if found {
			return fmt.Errorf("%w: duplicate %q entries", ErrUnsafeArchive, want)
		}
		if zf.UncompressedSize64 > uint64(maxBytes) {
			return fmt.Errorf("%w: %q is %d bytes, limit %d", ErrUnsafeArchive, zf.Name, zf.UncompressedSize64, maxBytes)
		}

		rc, err := zf.Open()
		if err != nil {
			return fmt.Errorf("%w: %v", ErrUnsafeArchive, err)
		}
		err = writeLimited(dst, rc, maxBytes)
		_ = rc.Close()
		if err != nil {
			return err
		}
		found = true
	}

	if !found {
		return ErrBinaryNotFound
	}
	return nil
}

// writeLimited writes at most maxBytes from r to a new file at dst (mode 0700 until verified).
// The declared size in an archive header is attacker-controlled, so the real byte count is enforced too.
func writeLimited(dst string, r io.Reader, maxBytes int64) error {
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o700)
	if err != nil {
		return err
	}

	n, copyErr := io.Copy(out, io.LimitReader(r, maxBytes+1))
	closeErr := out.Close()

	switch {
	case copyErr != nil:
		_ = os.Remove(dst)
		return fmt.Errorf("extracting binary: %w", copyErr)
	case n > maxBytes:
		_ = os.Remove(dst)
		return fmt.Errorf("%w: extracted size exceeds %d bytes", ErrUnsafeArchive, maxBytes)
	case closeErr != nil:
		_ = os.Remove(dst)
		return closeErr
	}
	return nil
}
