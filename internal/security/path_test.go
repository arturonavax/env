package security_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/arturonavax/env/internal/security"
)

func TestSafePath(t *testing.T) {
	baseDir := "/home/user/app"

	// Valid child path
	valid, err := security.SafePath(baseDir, "configs/app.yaml")
	if err != nil {
		t.Fatalf("expected valid path, got error: %v", err)
	}
	expected := filepath.Clean("/home/user/app/configs/app.yaml")
	if valid != expected {
		t.Fatalf("expected %s, got %s", expected, valid)
	}

	// Path traversal attack
	_, err = security.SafePath(baseDir, "../../etc/passwd")
	if err == nil {
		t.Fatalf("expected path traversal error, got nil")
	}
	if !strings.Contains(err.Error(), "path traversal detected") {
		t.Fatalf("expected path traversal message, got: %v", err)
	}
}

func TestRedactSensitives(t *testing.T) {
	input := "API_KEY=secret_token_value_1234\nNormal text line\nPASSWORD: mysecretpassword\n"
	redacted := security.RedactSensitives(input)

	if strings.Contains(redacted, "secret_token_value_1234") {
		t.Fatalf("expected secret token to be redacted")
	}
	if strings.Contains(redacted, "mysecretpassword") {
		t.Fatalf("expected password to be redacted")
	}
	if !strings.Contains(redacted, "Normal text line") {
		t.Fatalf("expected normal line to remain")
	}
}
