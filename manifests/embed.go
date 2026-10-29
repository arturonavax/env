package manifests

import "embed"

// EmbeddedFS bundles the canonical declarative YAML manifests into the anx binary
// so standalone executions work without a repository checkout, while repo and user
// drop-in directories dynamically overlay these defaults without recompiling Go code.
//
//go:embed packages/*.yaml configs.yaml profiles.yaml
var EmbeddedFS embed.FS
