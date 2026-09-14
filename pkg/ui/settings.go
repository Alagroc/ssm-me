package ui

import (
	"github.com/Alagroc/ssm-me/pkg/debuglog"
	"github.com/Alagroc/ssm-me/pkg/store"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SettingsView struct {
	app                 *App
	root                *tview.Flex
	form                *tview.Form
	debugLogCheckbox    *tview.Checkbox
	autoRefreshCheckbox *tview.Checkbox
	help                *tview.TextView

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

	// Checkboxes are built standalone (not via Form.AddCheckbox) and styled
	// explicitly in applyTheme, rather than left to Form's generic
	// SetFieldBackgroundColor/SetFieldTextColor: those cross-wire a
	// checkbox's focus style (its SetFieldBackgroundColor sets focusStyle's
	// FOREGROUND, and SetFieldTextColor sets focusStyle's BACKGROUND) in a
	// way that produced checked/focused states with too little contrast to
	// read against the default theme's background.
	v.debugLogCheckbox = tview.NewCheckbox().
		SetLabel("Enable debug log").
		SetChecked(debuglog.Enabled()).
		SetChangedFunc(func(checked bool) {
			app.toggleDebugLog(checked)
		})
	v.form.AddFormItem(v.debugLogCheckbox)

	settings, _ := store.LoadSettings()
	v.autoRefreshCheckbox = tview.NewCheckbox().
		SetLabel("Auto-refresh nodes on startup").
		SetChecked(settings.AutoRefresh).
		SetChangedFunc(func(checked bool) {
			app.toggleAutoRefresh(checked)
		})
	v.form.AddFormItem(v.autoRefreshCheckbox)

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

	// Checked = solid accent block (the theme's brightest color) so "on" is
	// unmistakable; unchecked = border color, dimmer by contrast; focused =
	// fixed white-on-black regardless of theme, for a reliable cursor.
	for _, cb := range []*tview.Checkbox{v.debugLogCheckbox, v.autoRefreshCheckbox} {
		cb.SetCheckedStyle(tcell.StyleDefault.Background(activeTheme.Accent).Foreground(activeTheme.Background))
		cb.SetUncheckedStyle(tcell.StyleDefault.Background(activeTheme.Border).Foreground(activeTheme.Background))
		cb.SetActivatedStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
		cb.SetCheckedString("X").SetUncheckedString(" ")
	}

	v.help.SetBackgroundColor(activeTheme.Background)
	v.updateHelp()
}
