#!/bin/bash
# Run: curl -fsSL "https://env.arturonavax.dev/install_golang_tools.sh" | bash
if [[ "$(command -v go)" == "" ]]; then
	echo "go is not installed on this system."

	exit 1
fi

go install golang.org/x/tools/gopls@latest
go install github.com/go-delve/delve/cmd/dlv@latest
go install github.com/golang/vscode-go/vscgo@latest
go install github.com/haya14busa/goplay/cmd/goplay@latest
go install github.com/fatih/gomodifytags@latest
go install github.com/josharian/impl@latest
go install github.com/cweill/gotests/...@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install mvdan.cc/gofumpt@latest
go install github.com/segmentio/golines@latest
go install golang.org/x/tools/cmd/goimports@latest
go install golang.org/x/tools/cmd/gorename@latest
go install github.com/koron/iferr@latest
go install golang.org/x/tools/cmd/callgraph@latest
go install golang.org/x/tools/cmd/guru@latest
go install github.com/davidrjenni/reftools/cmd/fillstruct@latest
go install github.com/davidrjenni/reftools/cmd/fillswitch@latest
go install github.com/onsi/ginkgo/v2/ginkgo@latest
go install github.com/kyoh86/richgo@latest
go install gotest.tools/gotestsum@latest
go install go.uber.org/mock/mockgen@latest
go install github.com/tmc/json-to-struct@latest
go install github.com/abenz1267/gomvp@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/abice/go-enum@latest
go install golang.org/x/tools/cmd/gonew@latest
