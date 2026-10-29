package hacks

import "embed"

// EmbeddedFS bundles canonical declarative workstation hack manifests and scripts into the anx binary
// so standalone executions work without a repository checkout, while live <repo>/hacks/*/hack.yaml
// and user ~/.config/anx/hacks.d/*.yaml directories dynamically overlay these definitions.
//
//go:embed */hack.yaml */*.sh
var EmbeddedFS embed.FS
