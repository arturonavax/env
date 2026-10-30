package configsync_test

import (
	"strings"
	"testing"

	"github.com/arturonavax/env/internal/configsync"
)

func TestStripJSONCComments(t *testing.T) {
	jsonc := `{
		// This is a comment
		"key": "value // not a comment",
		/* Multi-line
		   comment */
		"number": 42
	}`

	clean := configsync.StripJSONCComments(jsonc)
	if !strings.Contains(clean, `"key": "value // not a comment"`) {
		t.Fatalf("expected string containing // to be preserved, got: %s", clean)
	}
	if strings.Contains(clean, "This is a comment") || strings.Contains(clean, "Multi-line") {
		t.Fatalf("expected comments to be removed, got: %s", clean)
	}
}

func TestSemanticEqualityJSON(t *testing.T) {
	a := `{"b": 2, "a": 1}`
	b := `{
		// Comments
		"a": 1,
		"b": 2
	}`

	equal, err := configsync.IsSemanticEqual("config.jsonc", a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equal {
		t.Fatalf("expected semantically identical JSON/JSONC to be equal")
	}

	c := `{"a": 1, "b": 3}`
	equalDiff, err := configsync.IsSemanticEqual("config.json", a, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if equalDiff {
		t.Fatalf("expected different JSON to be non-equal")
	}
}

func TestSemanticEqualityTOML(t *testing.T) {
	a := `# Header
[section]
foo = "bar"
num = 10
`
	b := `[section]
num = 10
foo = "bar" # inline comment
`

	equal, err := configsync.IsSemanticEqual("starship.toml", a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equal {
		t.Fatalf("expected semantically identical TOML to be equal")
	}

	c := `[section]
foo = "other"
`
	equalDiff, err := configsync.IsSemanticEqual("starship.toml", a, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if equalDiff {
		t.Fatalf("expected different TOML to be non-equal")
	}
}
