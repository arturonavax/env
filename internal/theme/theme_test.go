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

	// Verify Tokyo Night Dark uses red accent (#f7768e)
	tokyoNight := theme.ByName("tokyo-night")
	if tokyoNight.Accent != tokyoNight.Red {
		t.Fatalf("expected tokyo-night accent to be red (%v), got %v", tokyoNight.Red, tokyoNight.Accent)
	}

	tokyoNightStorm := theme.ByName("tokyo-night-storm")
	if tokyoNightStorm.Accent != tokyoNightStorm.Red {
		t.Fatalf("expected tokyo-night-storm accent to be red (%v), got %v", tokyoNightStorm.Red, tokyoNightStorm.Accent)
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
