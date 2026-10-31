package security

import (
	"strings"
)

// TrustLevel classifies how safe a directory is to act as the origin of manifests and dotfiles.
//
// The zero value is TrustUnknown on purpose: a verifier that cannot prove safety must never
// be read as "trusted" by accident (fail-closed).
type TrustLevel int

const (
	// TrustUnknown means the verifier could not establish the directory's safety.
	TrustUnknown TrustLevel = iota
	// TrustTrusted means only the current user and privileged system principals can modify the directory.
	TrustTrusted
	// TrustDegraded means broad-but-authenticated principals (e.g. Windows "Users", a shared POSIX group)
	// can write. Mutations are allowed but the condition is surfaced to the user.
	TrustDegraded
	// TrustUntrusted means unprivileged or anonymous principals can modify the directory, or it
	// belongs to another user. Mutations are refused.
	TrustUntrusted
)

// String returns a stable lowercase label for logs, JSON and UI badges.
func (l TrustLevel) String() string {
	switch l {
	case TrustTrusted:
		return "trusted"
	case TrustDegraded:
		return "degraded"
	case TrustUntrusted:
		return "untrusted"
	default:
		return "unknown"
	}
}

// Blocks reports whether mutating operations must be refused at this level.
func (l TrustLevel) Blocks() bool {
	return l == TrustUnknown || l == TrustUntrusted
}

// TrustReport is the outcome of a directory trust evaluation.
type TrustReport struct {
	Path    string     `json:"path"`
	Level   TrustLevel `json:"-"`
	Label   string     `json:"level"`
	Reasons []string   `json:"reasons,omitempty"`
}

// Reason joins all findings into a single human-readable sentence.
func (r TrustReport) Reason() string {
	return strings.Join(r.Reasons, "; ")
}

// newTrustReport builds a report and keeps Label in sync with Level.
func newTrustReport(path string, level TrustLevel, reasons ...string) TrustReport {
	return TrustReport{Path: path, Level: level, Label: level.String(), Reasons: reasons}
}

// worse returns the more severe of two levels, treating Unknown as worse than Degraded
// but less severe than Untrusted.
func worse(a, b TrustLevel) TrustLevel {
	return max(severity(a), severity(b)).level()
}

type severityRank int

func (s severityRank) level() TrustLevel {
	switch s {
	case 3:
		return TrustUntrusted
	case 2:
		return TrustUnknown
	case 1:
		return TrustDegraded
	default:
		return TrustTrusted
	}
}

func severity(l TrustLevel) severityRank {
	switch l {
	case TrustUntrusted:
		return 3
	case TrustUnknown:
		return 2
	case TrustDegraded:
		return 1
	default:
		return 0
	}
}

// DirectoryTrustVerifier evaluates whether a directory can be trusted as a manifest or dotfile origin.
// Implementations are OS specific: POSIX mode bits are not a valid proxy for NTFS ACLs.
type DirectoryTrustVerifier interface {
	VerifyDirectory(path string) TrustReport
}

// DirectoryTrustVerifierFunc adapts a function to DirectoryTrustVerifier (handy in tests).
type DirectoryTrustVerifierFunc func(path string) TrustReport

// VerifyDirectory implements DirectoryTrustVerifier.
func (f DirectoryTrustVerifierFunc) VerifyDirectory(path string) TrustReport { return f(path) }

// DefaultTrustVerifier returns the verifier for the running operating system.
func DefaultTrustVerifier() DirectoryTrustVerifier {
	return platformTrustVerifier{}
}
