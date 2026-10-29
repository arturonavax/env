package theme_test

import (
	"testing"

	"github.com/arturonavax/env/internal/theme"
)

func TestTokyoNightTokens(t *testing.T) {
	styles := theme.DefaultStyles()
	if styles.Palette.Name != "tokyo-night" {
		t.Fatalf("expected tokyo-night, got %s", styles.Palette.Name)
	}

	rendered := styles.Header.Render("anx dashboard")
	if rendered == "" {
		t.Fatalf("expected rendered header, got empty string")
	}

	successRender := styles.Success.Render("Installed")
	if successRender == "" {
		t.Fatalf("expected rendered success, got empty string")
	}
}
