package selfupdate

import (
	"runtime/debug"
	"strings"
)

// Origin records how the running binary was produced, which decides how it may update itself.
type Origin string

const (
	// OriginRelease is an official release artifact stamped by the release pipeline.
	OriginRelease Origin = "release"
	// OriginSource is a local build made by the bootstrap installer from a repository checkout.
	OriginSource Origin = "source"
	// OriginGoInstall is a binary produced by `go install module/cmd/anx@version`.
	OriginGoInstall Origin = "go-install"
	// OriginDev is an unstamped build (go run, go build, tests).
	OriginDev Origin = "dev"
)

// Module and repository coordinates used for updates and `go install`.
const (
	ModulePath = "github.com/arturonavax/env"
	CmdPackage = ModulePath + "/cmd/anx"
	Repository = "arturonavax/env"
)

// Build-time stamps. Set with:
//
//	-X github.com/arturonavax/env/internal/selfupdate.origin=release
//	-X github.com/arturonavax/env/internal/selfupdate.releasePublicKey=<base64 ed25519 public key>
//
// They live here (not in package main) so the lifecycle code can read them without plumbing.
var (
	origin           string
	releasePublicKey string
)

// BuildInfo is the resolved provenance of the running binary.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
	Origin  Origin
}

// readBuildInfo is swapped in tests.
var readBuildInfo = debug.ReadBuildInfo

// Resolve merges linker-injected values with the module metadata Go embeds in every binary.
//
// `go install module@v1.2.3` cannot run -ldflags, so without this the binary would keep a stale
// hard-coded version and the updater would believe it is current forever.
func Resolve(version, commit, date string) BuildInfo {
	info := BuildInfo{
		Version: strings.TrimSpace(version),
		Commit:  strings.TrimSpace(commit),
		Date:    strings.TrimSpace(date),
		Origin:  parseOrigin(origin),
	}

	if info.Origin == OriginRelease || info.Origin == OriginSource {
		info.fillDefaults()
		return info
	}

	if bi, ok := readBuildInfo(); ok {
		if v := bi.Main.Version; v != "" && v != "(devel)" {
			info.Origin = OriginGoInstall
			info.Version = strings.TrimSuffix(v, "+incompatible")
		}
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				if info.Commit == "" || info.Commit == "dev" {
					info.Commit = shortRev(s.Value)
				}
			case "vcs.time":
				if info.Date == "" {
					info.Date = s.Value
				}
			}
		}
	}

	if info.Origin == "" {
		info.Origin = OriginDev
	}
	info.fillDefaults()
	return info
}

func (b *BuildInfo) fillDefaults() {
	if b.Version == "" {
		b.Version = "dev"
	}
	if b.Commit == "" {
		b.Commit = "dev"
	}
}

func parseOrigin(s string) Origin {
	switch o := Origin(strings.ToLower(strings.TrimSpace(s))); o {
	case OriginRelease, OriginSource:
		return o
	}
	return ""
}

func shortRev(rev string) string {
	const n = 7
	if len(rev) > n {
		return rev[:n]
	}
	return rev
}
