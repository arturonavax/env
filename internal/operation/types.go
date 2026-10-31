package operation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// OperationKind identifies the specific typed operation in the IR.
type OperationKind string

const (
	OpExecArgv             OperationKind = "exec_argv"
	OpDownloadVerified     OperationKind = "download_verified"
	OpEnsureDirectory      OperationKind = "ensure_directory"
	OpCopyAtomic           OperationKind = "copy_atomic"
	OpWriteAtomic          OperationKind = "write_atomic"
	OpDeleteOwned          OperationKind = "delete_owned"
	OpCreateSymlinkOwned   OperationKind = "create_symlink_owned"
	OpPackageInstall       OperationKind = "package_install"
	OpPackageRemove        OperationKind = "package_remove"
	OpPackageUpgrade       OperationKind = "package_upgrade"
	OpGitFetchPinned       OperationKind = "git_fetch_pinned"
	OpBuildFromSource      OperationKind = "build_from_source"
	OpVerifyChecksum       OperationKind = "verify_checksum"
	OpConfigMergeThreeWay  OperationKind = "config_merge_3way"
	OpSnapshotCreate       OperationKind = "snapshot_create"
	OpSnapshotRestore      OperationKind = "snapshot_restore"
	OpRecordState          OperationKind = "record_state"
	OpCompensate           OperationKind = "compensate"
	OpAnxRuntimeDelegation OperationKind = "anx_runtime_delegation"
)

// SideEffectClass categorizes the scope of mutation.
type SideEffectClass string

const (
	SideEffectNone          SideEffectClass = "none"
	SideEffectFilesystem    SideEffectClass = "filesystem"
	SideEffectSystemPackage SideEffectClass = "system_package"
	SideEffectProcess       SideEffectClass = "process"
)

// RiskLevel indicates the blast radius of an operation.
type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// Condition defines a check that must pass before (precondition) or after (postcondition) execution.
type Condition struct {
	Type     string `json:"type"` // "file_exists", "file_missing", "hash_equals", "command_succeeds"
	Target   string `json:"target"`
	Expected string `json:"expected,omitempty"`
}

// Operation defines a single typed, atomic, verifiable unit of execution.
type Operation struct {
	OpID               string          `json:"op_id"`
	Kind               OperationKind   `json:"kind"`
	Description        string          `json:"description"`
	Resources          []string        `json:"resources,omitempty"`
	DependsOn          []string        `json:"depends_on,omitempty"`
	TargetOS           string          `json:"target_os,omitempty"`
	TargetArch         string          `json:"target_arch,omitempty"`
	ProviderID         string          `json:"provider_id,omitempty"`
	Argv               []string        `json:"argv,omitempty"`
	Env                []string        `json:"env,omitempty"`
	Cwd                string          `json:"cwd,omitempty"`
	Mode               uint32          `json:"mode,omitempty"`
	Content            string          `json:"content,omitempty"`
	SourcePath         string          `json:"source_path,omitempty"`
	TargetPath         string          `json:"target_path,omitempty"`
	ExpectedBeforeHash string          `json:"expected_before_hash,omitempty"`
	ExpectedAfterHash  string          `json:"expected_after_hash,omitempty"`
	Preconditions      []Condition     `json:"preconditions,omitempty"`
	Postconditions     []Condition     `json:"postconditions,omitempty"`
	SideEffect         SideEffectClass `json:"side_effect"`
	Ownership          string          `json:"ownership,omitempty"`
	IdempotencyKey     string          `json:"idempotency_key,omitempty"`
	Risk               RiskLevel       `json:"risk"`
	RollbackPolicy     string          `json:"rollback_policy,omitempty"`
	Provenance         string          `json:"provenance,omitempty"`
}

// PlanIR represents an immutable, DAG-ordered sequence of typed operations.
type PlanIR struct {
	ID            string      `json:"id"`
	SchemaVersion int         `json:"schema_version"`
	Operations    []Operation `json:"operations"`
}

// CanonicalDigest computes the deterministic SHA-256 digest of the canonical JSON representation.
func (p *PlanIR) CanonicalDigest() (string, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("marshaling canonical plan IR: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
