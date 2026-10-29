package platform_test

import (
	"runtime"
	"testing"

	"github.com/arturonavax/env/internal/platform"
)

func TestPlatformDetection(t *testing.T) {
	info := platform.Detect()

	if info.OS() != runtime.GOOS {
		t.Fatalf("expected OS %s, got %s", runtime.GOOS, info.OS())
	}
	if info.Arch() != runtime.GOARCH {
		t.Fatalf("expected Arch %s, got %s", runtime.GOARCH, info.Arch())
	}
	if info.HomeDir() == "" {
		t.Fatalf("expected non-empty HomeDir")
	}
	if info.ConfigDir() == "" {
		t.Fatalf("expected non-empty ConfigDir")
	}
	if info.StateDir() == "" {
		t.Fatalf("expected non-empty StateDir")
	}
}
