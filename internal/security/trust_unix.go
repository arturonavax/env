//go:build !windows

package security

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// platformTrustVerifier implements DirectoryTrustVerifier with POSIX owner and mode semantics.
//
// Policy (leaf directory and every ancestor up to "/"):
//   - the entry must be owned by the effective user or root; anything else lets a third party
//     rename or replace the tree;
//   - other-writable is untrusted unless the sticky bit is set on an ancestor (e.g. /tmp);
//   - group-writable is untrusted when the group is not one of the caller's groups; it is
//     degraded when the caller belongs to a non-primary shared group.
type platformTrustVerifier struct{}

// VerifyDirectory implements DirectoryTrustVerifier.
func (platformTrustVerifier) VerifyDirectory(path string) TrustReport {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return newTrustReport(path, TrustUnknown, fmt.Sprintf("cannot resolve %q: %v", path, err))
	}

	uid := uint32(os.Geteuid())
	gids := callerGroups()

	level := TrustTrusted
	var reasons []string

	for current, leaf := resolved, true; ; leaf = false {
		info, statErr := os.Lstat(current)
		if statErr != nil {
			return newTrustReport(path, TrustUnknown, fmt.Sprintf("cannot stat %q: %v", current, statErr))
		}
		if leaf && !info.IsDir() {
			return newTrustReport(path, TrustUntrusted, fmt.Sprintf("%q is not a directory", current))
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return newTrustReport(path, TrustUnknown, fmt.Sprintf("no ownership data for %q", current))
		}

		entryLevel, entryReasons := evaluateEntry(current, info.Mode(), st, uid, gids, leaf)
		level = worse(level, entryLevel)
		reasons = append(reasons, entryReasons...)

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return newTrustReport(path, level, reasons...)
}

func evaluateEntry(
	path string,
	mode os.FileMode,
	st *syscall.Stat_t,
	uid uint32,
	gids map[uint32]struct{},
	leaf bool,
) (TrustLevel, []string) {
	level := TrustTrusted
	var reasons []string

	if st.Uid != uid && st.Uid != 0 {
		level = TrustUntrusted
		reasons = append(reasons, fmt.Sprintf("%q is owned by uid %d", path, st.Uid))
	}

	perm := mode.Perm()
	sticky := mode&os.ModeSticky != 0

	if perm&0o002 != 0 && (leaf || !sticky) {
		level = TrustUntrusted
		reasons = append(reasons, fmt.Sprintf("%q is world-writable (mode %04o)", path, perm))
	}

	if perm&0o020 != 0 && !(sticky && !leaf) {
		_, member := gids[st.Gid]
		switch {
		case !member:
			level = worse(level, TrustUntrusted)
			reasons = append(reasons, fmt.Sprintf("%q is writable by foreign group gid %d", path, st.Gid))
		case leaf && st.Gid != uint32(os.Getegid()):
			level = worse(level, TrustDegraded)
			reasons = append(reasons, fmt.Sprintf("%q is writable by shared group gid %d", path, st.Gid))
		}
	}

	return level, reasons
}

// callerGroups returns the numeric groups of the current process without touching the user database,
// which keeps it working in minimal containers.
func callerGroups() map[uint32]struct{} {
	groups := map[uint32]struct{}{uint32(os.Getegid()): {}}
	if supplementary, err := os.Getgroups(); err == nil {
		for _, g := range supplementary {
			groups[uint32(g)] = struct{}{}
		}
	}
	return groups
}
