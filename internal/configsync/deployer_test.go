package configsync_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arturonavax/env/internal/configsync"
)

func TestDeployerAtomicCopy(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "anx-deployer-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	src := filepath.Join(tempDir, "source.txt")
	err = os.WriteFile(src, []byte("atomic deploy test"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(tempDir, "dest", "target.txt")
	deployer := configsync.NewDeployer()

	err = deployer.Deploy(src, dst, configsync.ModeCopy)
	if err != nil {
		t.Fatalf("expected deploy to succeed, got: %v", err)
	}

	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "atomic deploy test" {
		t.Fatalf("expected 'atomic deploy test', got '%s'", string(content))
	}
}
