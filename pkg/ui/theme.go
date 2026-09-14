package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Theme is the small set of semantic colors applied across the UI's chrome
// (headers, borders, hints, selection, secondary info). AccentTag/InfoTag
// are the matching tview color-tag names, used to build "[tag]...[-]"
// markup at render time. Semantic status colors (node Ready/NotReady,
// execution Success/Failed, and one-off status-bar messages) are NOT
// themed — they stay red/green/yellow regardless, since that's meaningful
// independent of skin.
type Theme struct {
	Name       string
	Label      string
	Background tcell.Color
	Border     tcell.Color
	Accent     tcell.Color
	AccentTag  string
	Info       tcell.Color
	InfoTag    string
}

var themeOrder = []string{"default", "dark", "high-contrast"}

var themes = map[string]Theme{
	"default": {
		Name: "default", Label: "Default (blue)",
		Background: tcell.ColorNavy, Border: tcell.ColorAqua,
		Accent: tcell.ColorYellow, AccentTag: "yellow",
		Info: tcell.ColorAqua, InfoTag: "aqua",
	},
	"dark": {
		Name: "dark", Label: "Dark",
		Background: tcell.ColorBlack, Border: tcell.ColorSteelBlue,
		Accent: tcell.ColorDodgerBlue, AccentTag: "dodgerblue",
		Info: tcell.ColorGray, InfoTag: "gray",
	},
	"high-contrast": {
		Name: "high-contrast", Label: "High Contrast",
		Background: tcell.ColorBlack, Border: tcell.ColorWhite,
		Accent: tcell.ColorYellow, AccentTag: "yellow",
		Info: tcell.ColorWhite, InfoTag: "white",
	},
}

// activeTheme is the currently applied theme. There's only ever one App
// instance in this program, so a package-level var is simplest.
var activeTheme = themes["default"]

func themeLabels() []string {
	labels := make([]string, len(themeOrder))
	for i, n := range themeOrder {
		labels[i] = themes[n].Label
	}
	return labels
}

func themeIndex(name string) int {
	for i, n := range themeOrder {
		if n == name {
			return i
		}
	}
	return 0
}

// setTheme resolves name to a Theme (falling back to "default"), sets it as
// the active theme, and updates tview's global style defaults. tview.Box
// captures those at construction time, so for a brand-new primitive to
// pick them up, this must run before it's created; for primitives that
// already exist, their colors must be re-applied explicitly (see each
// view's applyTheme method).
func setTheme(name string) Theme {
	t, ok := themes[name]
	if !ok {
		t = themes["default"]
	}
	activeTheme = t

	tview.Styles.PrimitiveBackgroundColor = t.Background
	tview.Styles.ContrastBackgroundColor = tcell.ColorBlack
	tview.Styles.MoreContrastBackgroundColor = tcell.ColorTeal
	tview.Styles.BorderColor = t.Border
	tview.Styles.TitleColor = t.Accent
	tview.Styles.GraphicsColor = t.Border
	tview.Styles.PrimaryTextColor = tcell.ColorWhite
	tview.Styles.SecondaryTextColor = t.Accent
	tview.Styles.TertiaryTextColor = t.Info
	tview.Styles.InverseTextColor = t.Background
	tview.Styles.ContrastSecondaryTextColor = t.Accent

	return t
}

func accentTag(s string) string { return "[" + activeTheme.AccentTag + "]" + s + "[-]" }
func infoTag(s string) string   { return "[" + activeTheme.InfoTag + "]" + s + "[-]" }
