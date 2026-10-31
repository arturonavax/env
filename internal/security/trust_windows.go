//go:build windows

package security

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Access-mask bits that let a principal change or replace a directory tree.
// Values are the documented winnt.h constants; GENERIC_* are included because
// DACLs written by older tools may still carry unmapped generic rights.
const (
	maskAddFile        = 0x00000002 // FILE_ADD_FILE / FILE_WRITE_DATA
	maskAddSubdir      = 0x00000004 // FILE_ADD_SUBDIRECTORY / FILE_APPEND_DATA
	maskDeleteChild    = 0x00000040 // FILE_DELETE_CHILD
	maskDelete         = 0x00010000
	maskWriteDAC       = 0x00040000
	maskWriteOwner     = 0x00080000
	maskGenericAll     = 0x10000000
	maskGenericWrite   = 0x40000000
	maskLeafWriteBits  = maskAddFile | maskAddSubdir | maskDeleteChild | maskDelete | maskWriteDAC | maskWriteOwner | maskGenericAll | maskGenericWrite
	maskRenameChildren = maskDeleteChild | maskWriteDAC | maskWriteOwner | maskGenericAll
)

// platformTrustVerifier implements DirectoryTrustVerifier with real NTFS owner and DACL analysis.
//
// POSIX mode bits reported by Go on Windows are synthetic (0777/0666) and say nothing about ACLs,
// so they are never consulted here. Policy:
//   - the owner must be the current user, Administrators, SYSTEM or TrustedInstaller;
//   - a NULL DACL or an allow ACE for Everyone/Anonymous/Guests with write rights is untrusted;
//   - an allow ACE for Authenticated Users, Users, Interactive or any other non-privileged
//     principal with write rights is degraded (shared write, often inherited from a drive root);
//   - inherit-only ACEs do not apply to the object itself and are ignored;
//   - deny ACEs are honoured per SID;
//   - reparse points are resolved first, and the resolved target is the one evaluated.
//
// If any Win32 call fails the result is TrustUnknown, never an implicit "trusted".
type platformTrustVerifier struct{}

type sidClass int

const (
	classOtherPrincipal sidClass = iota
	classPrivileged
	classBroadAuthenticated
	classWorld
)

// VerifyDirectory implements DirectoryTrustVerifier.
func (platformTrustVerifier) VerifyDirectory(path string) TrustReport {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return newTrustReport(path, TrustUnknown, fmt.Sprintf("cannot resolve %q: %v", path, err))
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return newTrustReport(path, TrustUnknown, fmt.Sprintf("cannot stat %q: %v", resolved, err))
	}
	if !info.IsDir() {
		return newTrustReport(path, TrustUntrusted, fmt.Sprintf("%q is not a directory", resolved))
	}

	user, err := currentUserSID()
	if err != nil {
		return newTrustReport(path, TrustUnknown, fmt.Sprintf("cannot read current user token: %v", err))
	}

	level := TrustTrusted
	var reasons []string

	for current, leaf := resolved, true; ; leaf = false {
		entryLevel, entryReasons := evaluateWindowsEntry(current, user, leaf)
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

func evaluateWindowsEntry(path string, user *windows.SID, leaf bool) (TrustLevel, []string) {
	sd, err := windows.GetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		return TrustUnknown, []string{fmt.Sprintf("cannot read security descriptor of %q: %v", path, err)}
	}

	level := TrustTrusted
	var reasons []string

	owner, _, err := sd.Owner()
	switch {
	case err != nil || owner == nil:
		level = worse(level, TrustUnknown)
		reasons = append(reasons, fmt.Sprintf("cannot read owner of %q", path))
	case !ownerIsTrusted(owner, user):
		level = worse(level, TrustUntrusted)
		reasons = append(reasons, fmt.Sprintf("%q is owned by %s", path, owner.String()))
	}

	dacl, _, err := sd.DACL()
	switch {
	case err != nil:
		level = worse(level, TrustUnknown)
		reasons = append(reasons, fmt.Sprintf("cannot read DACL of %q: %v", path, err))
	case dacl == nil:
		level = worse(level, TrustUntrusted)
		reasons = append(reasons, fmt.Sprintf("%q has a NULL DACL (full access for everyone)", path))
	default:
		aclLevel, aclReasons := evaluateDACL(path, dacl, user, leaf)
		level = worse(level, aclLevel)
		reasons = append(reasons, aclReasons...)
	}

	return level, reasons
}

func evaluateDACL(path string, dacl *windows.ACL, user *windows.SID, leaf bool) (TrustLevel, []string) {
	writeMask := uint32(maskRenameChildren)
	if leaf {
		writeMask = maskLeafWriteBits
	}

	level := TrustTrusted
	var reasons []string
	denied := map[string]uint32{}

	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			return worse(level, TrustUnknown), append(reasons, fmt.Sprintf("cannot read ACE %d of %q: %v", i, path, err))
		}
		if ace.Header.AceFlags&windows.INHERIT_ONLY_ACE != 0 {
			continue
		}

		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		key := sid.String()

		switch ace.Header.AceType {
		case windows.ACCESS_DENIED_ACE_TYPE:
			denied[key] |= uint32(ace.Mask)
		case windows.ACCESS_ALLOWED_ACE_TYPE:
			effective := uint32(ace.Mask) &^ denied[key] & writeMask
			if effective == 0 {
				continue
			}
			switch classifySID(sid, user) {
			case classWorld:
				level = worse(level, TrustUntrusted)
				reasons = append(reasons, fmt.Sprintf("%q grants write to world principal %s", path, key))
			case classBroadAuthenticated, classOtherPrincipal:
				level = worse(level, TrustDegraded)
				reasons = append(reasons, fmt.Sprintf("%q grants write to shared principal %s", path, key))
			case classPrivileged:
				// Current user, Administrators, SYSTEM, TrustedInstaller, CREATOR OWNER: expected.
			}
		}
	}

	return level, reasons
}

func classifySID(sid, user *windows.SID) sidClass {
	if sid.Equals(user) {
		return classPrivileged
	}
	for _, wk := range []windows.WELL_KNOWN_SID_TYPE{
		windows.WinWorldSid,
		windows.WinAnonymousSid,
		windows.WinBuiltinGuestsSid,
	} {
		if sid.IsWellKnown(wk) {
			return classWorld
		}
	}
	for _, wk := range []windows.WELL_KNOWN_SID_TYPE{
		windows.WinBuiltinAdministratorsSid,
		windows.WinLocalSystemSid,
		windows.WinCreatorOwnerSid,
		windows.WinNetworkServiceSid,
		windows.WinLocalServiceSid,
	} {
		if sid.IsWellKnown(wk) {
			return classPrivileged
		}
	}
	if isTrustedInstaller(sid) {
		return classPrivileged
	}
	for _, wk := range []windows.WELL_KNOWN_SID_TYPE{
		windows.WinAuthenticatedUserSid,
		windows.WinBuiltinUsersSid,
		windows.WinInteractiveSid,
		windows.WinNetworkSid,
	} {
		if sid.IsWellKnown(wk) {
			return classBroadAuthenticated
		}
	}
	return classOtherPrincipal
}

func ownerIsTrusted(owner, user *windows.SID) bool {
	return classifySID(owner, user) == classPrivileged
}

// isTrustedInstaller matches NT SERVICE\TrustedInstaller (S-1-5-80-956008885-...-2271478464).
func isTrustedInstaller(sid *windows.SID) bool {
	trustedInstaller, err := windows.StringToSid(
		"S-1-5-80-956008885-3418522649-1831038044-1853292631-2271478464",
	)
	return err == nil && sid.Equals(trustedInstaller)
}

func currentUserSID() (*windows.SID, error) {
	tokenUser, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	return tokenUser.User.Sid, nil
}
