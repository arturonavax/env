// Package selfupdate implements the ANX binary lifecycle: build provenance, release discovery,
// verified download, safe extraction and atomic replacement with rollback.
//
// The package has no dependency on the CLI layer; the `anx self update` command is a thin adapter.
package selfupdate

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Version is a parsed semantic version (https://semver.org). Build metadata is discarded
// because it never participates in precedence.
type Version struct {
	Major, Minor, Patch uint64
	Pre                 []string
}

// ErrInvalidVersion indicates a string that is not a semantic version.
var ErrInvalidVersion = errors.New("invalid semantic version")

// ParseVersion accepts "1.2.3", "v1.2.3", "v1.2.3-rc.1" and "v1.2.3+meta".
// Missing minor/patch components default to zero ("v1.2" == "v1.2.0").
func ParseVersion(s string) (Version, error) {
	raw := strings.TrimSpace(s)
	raw = strings.TrimPrefix(raw, "v")
	raw = strings.TrimSuffix(raw, "+incompatible")
	if i := strings.IndexByte(raw, '+'); i >= 0 {
		raw = raw[:i]
	}

	core, pre, hasPre := strings.Cut(raw, "-")
	parts := strings.Split(core, ".")
	if core == "" || len(parts) > 3 {
		return Version{}, fmt.Errorf("%w: %q", ErrInvalidVersion, s)
	}

	var nums [3]uint64
	for i, p := range parts {
		n, err := strconv.ParseUint(p, 10, 64)
		if err != nil || (len(p) > 1 && p[0] == '0') {
			return Version{}, fmt.Errorf("%w: %q", ErrInvalidVersion, s)
		}
		nums[i] = n
	}

	v := Version{Major: nums[0], Minor: nums[1], Patch: nums[2]}
	if hasPre {
		if pre == "" {
			return Version{}, fmt.Errorf("%w: empty pre-release in %q", ErrInvalidVersion, s)
		}
		v.Pre = strings.Split(pre, ".")
	}
	return v, nil
}

// String renders the canonical "vMAJOR.MINOR.PATCH[-pre]" form.
func (v Version) String() string {
	s := fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
	if len(v.Pre) > 0 {
		s += "-" + strings.Join(v.Pre, ".")
	}
	return s
}

// IsPrerelease reports whether the version carries a pre-release tag.
func (v Version) IsPrerelease() bool { return len(v.Pre) > 0 }

// Compare returns -1, 0 or +1 following semver precedence rules.
func (v Version) Compare(o Version) int {
	if c := cmpUint(v.Major, o.Major); c != 0 {
		return c
	}
	if c := cmpUint(v.Minor, o.Minor); c != 0 {
		return c
	}
	if c := cmpUint(v.Patch, o.Patch); c != 0 {
		return c
	}
	return comparePre(v.Pre, o.Pre)
}

func cmpUint(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// comparePre implements semver §11: a release outranks any pre-release of the same core; numeric
// identifiers rank below alphanumeric ones; a longer set outranks its own prefix.
func comparePre(a, b []string) int {
	switch {
	case len(a) == 0 && len(b) == 0:
		return 0
	case len(a) == 0:
		return 1
	case len(b) == 0:
		return -1
	}

	for i := 0; i < len(a) && i < len(b); i++ {
		if c := compareIdentifier(a[i], b[i]); c != 0 {
			return c
		}
	}
	return cmpUint(uint64(len(a)), uint64(len(b)))
}

func compareIdentifier(a, b string) int {
	an, aErr := strconv.ParseUint(a, 10, 64)
	bn, bErr := strconv.ParseUint(b, 10, 64)
	switch {
	case aErr == nil && bErr == nil:
		return cmpUint(an, bn)
	case aErr == nil:
		return -1
	case bErr == nil:
		return 1
	}
	return strings.Compare(a, b)
}
