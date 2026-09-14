package ui

import (
	"fmt"

	"github.com/Alagroc/ssm-me/pkg/awsclient"
	"github.com/Alagroc/ssm-me/pkg/debuglog"
	"github.com/Alagroc/ssm-me/pkg/kubectl"
	"github.com/Alagroc/ssm-me/pkg/store"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	pageNodes    = "nodes"
	pageExecute  = "execute"
	pageHistory  = "history"
	pageSettings = "settings"
)

type App struct {
	tv     *tview.Application
	pages  *tview.Pages
	header *tview.TextView
	status *tview.TextView

	nodes    []kubectl.Node
	selected map[string]bool
	context  string
	aws      *awsclient.Client

	nodesView    *NodesView
	executeView  *ExecuteView
	historyView  *HistoryView
	settingsView *SettingsView
}

func NewApp(aws *awsclient.Client, initialTheme string) *App {
	setTheme(initialTheme)

	a := &App{
		tv:       tview.NewApplication(),
		pages:    tview.NewPages(),
		header:   tview.NewTextView().SetDynamicColors(true),
		status:   tview.NewTextView().SetDynamicColors(true),
		selected: make(map[string]bool),
		aws:      aws,
	}

	a.nodesView = newNodesView(a)
	a.executeView = newExecuteView(a)
	a.historyView = newHistoryView(a)
	a.settingsView = newSettingsView(a)

	a.pages.AddPage(pageNodes, a.nodesView.root, true, true)
	a.pages.AddPage(pageExecute, a.executeView.root, true, false)
	a.pages.AddPage(pageHistory, a.historyView.root, true, false)
	a.pages.AddPage(pageSettings, a.settingsView.root, true, false)

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.header, 1, 0, false).
		AddItem(a.pages, 0, 1, true).
		AddItem(a.status, 1, 0, false)

	a.tv.SetRoot(layout, true)
	a.tv.SetInputCapture(a.globalKeys)

	a.renderHeader(pageNodes)
	if a.aws == nil {
		a.status.SetText(" [red]aws CLI not found — SSM features disabled[-]  " + infoTag("Press r to load nodes"))
	} else {
		a.status.SetText(" " + infoTag("Press r to load nodes"))
	}
	return a
}

func (a *App) Run() error {
	return a.tv.Run()
}

func (a *App) setStatus(msg string) {
	debuglog.Printf("status: %s", msg)
	a.tv.QueueUpdateDraw(func() {
		a.status.SetText(" " + msg)
	})
}

func (a *App) renderHeader(current string) {
	tabs := []struct{ page, label string }{
		{pageNodes, "1:Nodes"},
		{pageExecute, "2:Execute"},
		{pageHistory, "3:History"},
		{pageSettings, "4:Settings"},
	}
	h := " [::b]" + accentTag("ssm-me") + "[::-]  "
	for _, t := range tabs {
		if t.page == current {
			h += fmt.Sprintf("[black:%s] %s [-:-] ", activeTheme.AccentTag, t.label)
		} else {
			h += fmt.Sprintf("[white:-] %s [-:-] ", t.label)
		}
	}

	sel := 0
	for _, ok := range a.selected {
		if ok {
			sel++
		}
	}
	if sel > 0 {
		h += "  " + accentTag(fmt.Sprintf("%d selected", sel))
	}
	if a.context != "" {
		h += "  " + infoTag(fmt.Sprintf("ctx: %s", a.context))
	}
	a.header.SetText(h)
}

func (a *App) switchTo(page string) {
	a.pages.SwitchToPage(page)
	a.renderHeader(page)
	switch page {
	case pageExecute:
		a.executeView.update()
		a.tv.SetFocus(a.executeView.cmd)
	case pageNodes:
		a.tv.SetFocus(a.nodesView.table)
	case pageHistory:
		go a.historyView.refresh()
		a.tv.SetFocus(a.historyView.table)
	case pageSettings:
		a.tv.SetFocus(a.settingsView.root)
	}
}

// applyThemeLive switches the active theme, re-colors every already-built
// primitive (tview.Box only picks up tview.Styles at construction time),
// re-renders theme-tagged text, and persists the choice. It must be called
// synchronously from the main event-loop goroutine (e.g. a Form field's
// change callback) — it touches primitives directly with no locking, and
// calling setStatus/QueueUpdateDraw from in here would deadlock the loop
// against itself, the same way it would in any other input handler.
func (a *App) applyThemeLive(name string) {
	setTheme(name)

	a.header.SetBackgroundColor(activeTheme.Background)
	a.status.SetBackgroundColor(activeTheme.Background)

	a.nodesView.applyTheme()
	a.executeView.applyTheme()
	a.historyView.applyTheme()
	if a.settingsView != nil {
		a.settingsView.applyTheme()
	}

	current, _ := a.pages.GetFrontPage()
	a.renderHeader(current)

	a.status.SetText(" " + infoTag(fmt.Sprintf("Theme set to %s", activeTheme.Label)))

	if err := store.SaveSettings(store.Settings{Theme: name, DebugLog: debuglog.Enabled()}); err != nil {
		debuglog.Printf("save settings: %v", err)
	}
}

// toggleDebugLog enables/disables the debug log and persists the choice.
// Same main-goroutine-only constraint as applyThemeLive.
func (a *App) toggleDebugLog(enable bool) {
	path, err := debuglog.SetEnabled(enable)
	if err != nil {
		a.status.SetText(" [red]debug log: " + err.Error() + "[-]")
		return
	}
	if enable {
		a.status.SetText(" " + infoTag("Debug log enabled: "+path))
	} else {
		a.status.SetText(" " + infoTag("Debug log disabled"))
	}

	settings, err := store.LoadSettings()
	if err != nil {
		settings = store.Settings{Theme: activeTheme.Name}
	}
	settings.DebugLog = enable
	if err := store.SaveSettings(settings); err != nil {
		debuglog.Printf("save settings: %v", err)
	}
}

func (a *App) globalKeys(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyRune {
		switch event.Rune() {
		case '1':
			a.switchTo(pageNodes)
			return nil
		case '2':
			a.switchTo(pageExecute)
			return nil
		case '3':
			a.switchTo(pageHistory)
			return nil
		case '4':
			a.switchTo(pageSettings)
			return nil
		case 'Q':
			a.tv.Stop()
			return nil
		}
	}
	return event
}
