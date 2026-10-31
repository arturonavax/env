package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/arturonavax/env/internal/app"
	"github.com/arturonavax/env/internal/cli"
	"github.com/arturonavax/env/internal/platform"
)

func TestAuditCommandScriptMode(t *testing.T) {
	auditSvc := app.NewAuditService(nil, nil, nil)
	plat := platform.Detect()

	appCtx := &cli.AppContext{
		Platform: plat,
		AuditSvc: auditSvc,
		Viper:    viper.New(),
		Flags: cli.GlobalFlagState{
			Format: "text",
		},
	}

	rootCmd := cli.NewRootCommand(appCtx)
	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetArgs([]string{"audit", "--script", "--profile", "minimal"})

	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("audit --script failed: %v", err)
	}

	output := outBuf.String()
	if !strings.Contains(output, "#!/usr/bin/env bash") {
		t.Errorf("expected bash shebang in script output, got:\n%s", output)
	}
}
