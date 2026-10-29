package theme

import (
	"image/color"
	"os"

	"charm.land/lipgloss/v2"
)

// Palette holds the semantic color tokens for a theme.
type Palette struct {
	Name       string
	Background color.Color
	Foreground color.Color
	Accent     color.Color // Blue
	Cyan       color.Color
	Purple     color.Color
	Green      color.Color
	Yellow     color.Color
	Red        color.Color
	Muted      color.Color
	Border     color.Color
}

// TokyoNightPalette is the standard Tokyo Night dark palette.
var TokyoNightPalette = Palette{
	Name:       "tokyo-night",
	Background: lipgloss.Color("#1a1b26"),
	Foreground: lipgloss.Color("#c0caf5"),
	Accent:     lipgloss.Color("#7aa2f7"),
	Cyan:       lipgloss.Color("#7dcfff"),
	Purple:     lipgloss.Color("#bb9af7"),
	Green:      lipgloss.Color("#9ece6a"),
	Yellow:     lipgloss.Color("#e0af68"),
	Red:        lipgloss.Color("#f7768e"),
	Muted:      lipgloss.Color("#565f89"),
	Border:     lipgloss.Color("#3b4261"),
}

// TokyoNightStormPalette is the slightly darker Tokyo Night Storm palette.
var TokyoNightStormPalette = Palette{
	Name:       "tokyo-night-storm",
	Background: lipgloss.Color("#24283b"),
	Foreground: lipgloss.Color("#c0caf5"),
	Accent:     lipgloss.Color("#7aa2f7"),
	Cyan:       lipgloss.Color("#7dcfff"),
	Purple:     lipgloss.Color("#bb9af7"),
	Green:      lipgloss.Color("#9ece6a"),
	Yellow:     lipgloss.Color("#e0af68"),
	Red:        lipgloss.Color("#f7768e"),
	Muted:      lipgloss.Color("#565f89"),
	Border:     lipgloss.Color("#414868"),
}


// CatppuccinMochaPalette is the Catppuccin Mocha dark palette.
var CatppuccinMochaPalette = Palette{
	Name: "catppuccin", Background: lipgloss.Color("#1e1e2e"), Foreground: lipgloss.Color("#cdd6f4"),
	Accent: lipgloss.Color("#89b4fa"), Cyan: lipgloss.Color("#89dceb"), Purple: lipgloss.Color("#cba6f7"),
	Green: lipgloss.Color("#a6e3a1"), Yellow: lipgloss.Color("#f9e2af"), Red: lipgloss.Color("#f38ba8"),
	Muted: lipgloss.Color("#6c7086"), Border: lipgloss.Color("#45475a"),
}

// GruvboxPalette is the Gruvbox dark palette.
var GruvboxPalette = Palette{
	Name: "gruvbox", Background: lipgloss.Color("#282828"), Foreground: lipgloss.Color("#ebdbb2"),
	Accent: lipgloss.Color("#83a598"), Cyan: lipgloss.Color("#8ec07c"), Purple: lipgloss.Color("#d3869b"),
	Green: lipgloss.Color("#b8bb26"), Yellow: lipgloss.Color("#fabd2f"), Red: lipgloss.Color("#fb4934"),
	Muted: lipgloss.Color("#928374"), Border: lipgloss.Color("#504945"),
}

// NordPalette is the Nord palette.
var NordPalette = Palette{
	Name: "nord", Background: lipgloss.Color("#2e3440"), Foreground: lipgloss.Color("#d8dee9"),
	Accent: lipgloss.Color("#81a1c1"), Cyan: lipgloss.Color("#88c0d0"), Purple: lipgloss.Color("#b48ead"),
	Green: lipgloss.Color("#a3be8c"), Yellow: lipgloss.Color("#ebcb8b"), Red: lipgloss.Color("#bf616a"),
	Muted: lipgloss.Color("#616e88"), Border: lipgloss.Color("#434c5e"),
}

// NoColorPalette renders everything with the terminal default foreground (monochrome).
var NoColorPalette = Palette{
	Name: "no-color", Background: lipgloss.NoColor{}, Foreground: lipgloss.NoColor{},
	Accent: lipgloss.NoColor{}, Cyan: lipgloss.NoColor{}, Purple: lipgloss.NoColor{},
	Green: lipgloss.NoColor{}, Yellow: lipgloss.NoColor{}, Red: lipgloss.NoColor{},
	Muted: lipgloss.NoColor{}, Border: lipgloss.NoColor{},
}

var registry = []Palette{
	TokyoNightPalette, TokyoNightStormPalette, CatppuccinMochaPalette, GruvboxPalette, NordPalette, NoColorPalette,
}

// Names lists every available theme name in stable order.
func Names() []string {
	names := make([]string, 0, len(registry))
	for _, p := range registry {
		names = append(names, p.Name)
	}
	return names
}

// ByName resolves a palette by name; unknown names and NO_COLOR fall back safely.
func ByName(name string) Palette {
	if IsNoColor() {
		return NoColorPalette
	}
	for _, p := range registry {
		if p.Name == name {
			return p
		}
	}
	return TokyoNightPalette
}

// Styles encapsulates Lipgloss v2 styles derived from a Palette.
type Styles struct {
	Palette        Palette
	Header         lipgloss.Style
	TabActive      lipgloss.Style
	TabInactive    lipgloss.Style
	BorderActive   lipgloss.Style
	BorderInactive lipgloss.Style
	Success        lipgloss.Style
	Warning        lipgloss.Style
	Error          lipgloss.Style
	Muted          lipgloss.Style
	Highlight      lipgloss.Style
	SelectedRow    lipgloss.Style
}

// NewStyles creates a Styles instance from the specified palette.
func NewStyles(p Palette) *Styles {
	return &Styles{
		Palette: p,
		Header: lipgloss.NewStyle().
			Foreground(p.Accent).
			Bold(true),
		TabActive: lipgloss.NewStyle().
			Foreground(p.Background).
			Background(p.Accent).
			Bold(true).
			Padding(0, 1),
		TabInactive: lipgloss.NewStyle().
			Foreground(p.Muted).
			Padding(0, 1),
		BorderActive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.Accent),
		BorderInactive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.Border),
		Success: lipgloss.NewStyle().
			Foreground(p.Green).
			Bold(true),
		Warning: lipgloss.NewStyle().
			Foreground(p.Yellow).
			Bold(true),
		Error: lipgloss.NewStyle().
			Foreground(p.Red).
			Bold(true),
		Muted: lipgloss.NewStyle().
			Foreground(p.Muted),
		Highlight: lipgloss.NewStyle().
			Foreground(p.Cyan).
			Bold(true),
		SelectedRow: lipgloss.NewStyle().
			Foreground(p.Background).
			Background(p.Accent).
			Bold(true),
	}
}

// DefaultStyles returns the default Tokyo Night styles.
func DefaultStyles() *Styles {
	return NewStyles(TokyoNightPalette)
}

// IsNoColor checks if NO_COLOR environment variable is set.
func IsNoColor() bool {
	return os.Getenv("NO_COLOR") != ""
}
