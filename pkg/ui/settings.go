package ui

import (
	"github.com/Alagroc/ssm-me/pkg/debuglog"
	"github.com/Alagroc/ssm-me/pkg/store"
	"github.com/rivo/tview"
)

type SettingsView struct {
	app  *App
	root *tview.Flex
	form *tview.Form
	help *tview.TextView

	// initializing suppresses the dropdown's change callback while it's
	// being given its initial value during construction — Form.AddDropDown
	// wires the callback before applying the initial option, so it fires
	// immediately, before newSettingsView has returned and app.settingsView
	// has been assigned. Without this guard that's a nil-pointer panic.
	initializing bool
}

func newSettingsView(app *App) *SettingsView {
	v := &SettingsView{app: app, initializing: true}

	v.form = tview.NewForm()
	v.form.SetBorder(true)
	v.form.SetTitle(" Settings ")

	v.form.AddDropDown("Color scheme", themeLabels(), themeIndex(activeTheme.Name), func(_ string, index int) {
		if v.initializing {
			return
		}
		// Form callbacks run synchronously on the main event-loop goroutine,
		// same as any other input handler — call directly, don't spawn a
		// goroutine (that would race the draw loop) and don't route through
		// setStatus/QueueUpdateDraw (that would deadlock it against itself).
		app.applyThemeLive(themeOrder[index])
	})

	v.form.AddCheckbox("Enable debug log", debuglog.Enabled(), func(checked bool) {
		app.toggleDebugLog(checked)
	})

	settings, _ := store.LoadSettings()
	v.form.AddCheckbox("Auto-refresh nodes on startup", settings.AutoRefresh, func(checked bool) {
		app.toggleAutoRefresh(checked)
	})

	v.help = tview.NewTextView().SetDynamicColors(true)

	v.root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.form, 0, 1, true).
		AddItem(v.help, 1, 0, false)

	v.applyTheme()
	v.initializing = false
	return v
}

func (v *SettingsView) updateHelp() {
	v.help.SetText(" " + accentTag("Tab/Down") + ":next field  " + accentTag("Enter") + ":change  " +
		accentTag("◄►") + "/" + accentTag("1-4") + ":tabs")
}

// applyTheme re-colors this view's primitives after a live theme switch.
// Must run on the main event-loop goroutine (see App.applyThemeLive).
func (v *SettingsView) applyTheme() {
	v.form.SetBackgroundColor(activeTheme.Background)
	v.form.SetFieldBackgroundColor(activeTheme.Border)
	v.form.SetFieldTextColor(activeTheme.Background)
	v.form.SetLabelColor(activeTheme.Accent)
	v.form.SetButtonBackgroundColor(activeTheme.Border)
	v.form.SetButtonTextColor(activeTheme.Background)
	v.help.SetBackgroundColor(activeTheme.Background)
	v.updateHelp()
}
