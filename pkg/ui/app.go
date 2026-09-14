package ui

import (
	"fmt"

	"github.com/Alagroc/ssm-me/pkg/awsclient"
	"github.com/Alagroc/ssm-me/pkg/debuglog"
	"github.com/Alagroc/ssm-me/pkg/kubectl"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	pageNodes   = "nodes"
	pageExecute = "execute"
	pageHistory = "history"
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

	nodesView   *NodesView
	executeView *ExecuteView
	historyView *HistoryView
}

func NewApp(aws *awsclient.Client) *App {
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

	a.pages.AddPage(pageNodes, a.nodesView.root, true, true)
	a.pages.AddPage(pageExecute, a.executeView.root, true, false)
	a.pages.AddPage(pageHistory, a.historyView.root, true, false)

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.header, 1, 0, false).
		AddItem(a.pages, 0, 1, true).
		AddItem(a.status, 1, 0, false)

	a.tv.SetRoot(layout, true)
	a.tv.SetInputCapture(a.globalKeys)

	a.renderHeader(pageNodes)
	if a.aws == nil {
		a.status.SetText(" [red]aws CLI not found — SSM features disabled[-]  [gray]Press r to load nodes[-]")
	} else {
		a.status.SetText(" [gray]Press r to load nodes[-]")
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
	}
	h := " [::b][dodgerblue]ssm-me[-][::-]  "
	for _, t := range tabs {
		if t.page == current {
			h += fmt.Sprintf("[black:dodgerblue] %s [-:-] ", t.label)
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
		h += fmt.Sprintf("  [dodgerblue]%d selected[-]", sel)
	}
	if a.context != "" {
		h += fmt.Sprintf("  [gray]ctx: %s[-]", a.context)
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
		case 'Q':
			a.tv.Stop()
			return nil
		}
	}
	return event
}
