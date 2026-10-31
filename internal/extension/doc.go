// Package extension hosts out-of-process ANX extensions over a stdio JSON-RPC protocol.
//
// # Trust model (read this before relying on it)
//
// This package is NOT an operating-system sandbox. A launched extension runs with the full
// privileges of the invoking user: it can read and write that user's files, open network
// connections and spawn processes. Nothing here restricts the kernel-visible behaviour of the
// child once it is running.
//
// What the package does provide is a verifiable admission policy that is enforced BEFORE any
// byte of the extension executes:
//
//   - the binary must be pinned by SHA-256 and the on-disk content must match (ErrIntegrity);
//   - on Unix the binary and its directory must be owned by the current user or root and must not
//     be group/world-writable (ErrUntrustedBinary);
//   - the permissions the extension declares must be a subset of the permissions the host allows
//     (ErrPermissionDenied) — checked before launch, so an over-privileged extension never starts;
//   - the child receives a scrubbed environment (no inherited secrets), an empty private working
//     directory and, on Unix, its own process group so that Close terminates its whole tree;
//   - every protocol message is size-bounded and the handshake is time-bounded.
//
// Permissions are capability tokens enforced by the HOST when it mediates actions on behalf of an
// extension; they are not enforced on the child by the OS. After the handshake the extension may
// only claim permissions it declared up front, otherwise it is killed.
//
// Residual risk: there is a small window between hashing the binary and exec(2). It is narrowed by
// requiring a non-writable directory and by resolving symlinks before hashing, but it is not zero.
// Treat extensions as trusted code that was explicitly pinned by the operator. Real isolation
// (Landlock/seccomp, sandbox-exec, AppContainer) is a separate, platform-specific capability and
// is deliberately not claimed here.
package extension
