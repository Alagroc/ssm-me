package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/Alagroc/ssm-me/pkg/kubectl"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type NodesView struct {
	app      *App
	root     *tview.Flex
	table    *tview.Table
	filter   *tview.InputField
	info     *tview.TextView
	filtered []kubectl.Node
}

func newNodesView(app *App) *NodesView {
	v := &NodesView{app: app}

	v.filter = tview.NewInputField().
		SetLabel("Filter (key=val,...): ").
		SetFieldWidth(50)
	v.filter.SetDoneFunc(func(_ tcell.Key) {
		v.applyFilter()
		app.tv.SetFocus(v.table)
	})

	v.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)

	v.table.SetSelectedFunc(func(row, _ int) {
		if row == 0 {
			return
		}
		ref := v.table.GetCell(row, 0).GetReference()
		if ref == nil {
			return
		}
		name := ref.(string)
		app.selected[name] = !app.selected[name]
		v.renderRows()
		app.tv.QueueUpdateDraw(func() { app.renderHeader(pageNodes) })
	})

	v.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case event.Key() == tcell.KeyRune && event.Rune() == 'r':
			go v.refresh()
			return nil
		case event.Key() == tcell.KeyRune && (event.Rune() == 'f' || event.Rune() == '/'):
			app.tv.SetFocus(v.filter)
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == 'e':
			app.switchTo(pageExecute)
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == 't':
			go v.showTopStats()
			return nil
		case event.Key() == tcell.KeyEsc:
			app.selected = make(map[string]bool)
			v.renderRows()
			app.tv.QueueUpdateDraw(func() { app.renderHeader(pageNodes) })
			return nil
		}
		return event
	})

	v.info = tview.NewTextView().SetDynamicColors(true)

	help := tview.NewTextView().
		SetDynamicColors(true).
		SetText(" [yellow]Space/Enter[-]:select  [yellow]e[-]:execute  [yellow]r[-]:refresh  [yellow]f[-]:filter  [yellow]t[-]:top  [yellow]Esc[-]:clear selection  [yellow]Q[-]:quit")

	v.root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.filter, 1, 0, false).
		AddItem(v.table, 0, 1, true).
		AddItem(v.info, 1, 0, false).
		AddItem(help, 1, 0, false)

	v.renderHeader()
	return v
}

func (v *NodesView) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	v.app.setStatus("[yellow]Loading...[-]")

	if v.app.context == "" {
		if c, err := kubectl.GetCurrentContext(ctx); err == nil && c != "" {
			v.app.context = c
			v.app.tv.QueueUpdateDraw(func() { v.app.renderHeader(pageNodes) })
		}
	}

	nodes, err := kubectl.GetNodes(ctx)
	if err != nil {
		v.app.setStatus(fmt.Sprintf("[red]%v[-]", err))
		return
	}
	v.app.nodes = nodes
	v.applyFilter()
	v.app.setStatus(fmt.Sprintf("[green]%d nodes[-]", len(nodes)))
}

func (v *NodesView) applyFilter() {
	filters := kubectl.ParseFilters(v.filter.GetText())
	v.filtered = kubectl.FilterNodes(v.app.nodes, filters)
	v.app.tv.QueueUpdateDraw(func() {
		v.renderRows()
		v.info.SetText(fmt.Sprintf(" [gray]%d/%d nodes[-]", len(v.filtered), len(v.app.nodes)))
	})
}

func (v *NodesView) renderHeader() {
	cols := []struct {
		title string
		exp   int
	}{
		{"NAME", 3},
		{"STATUS", 0},
		{"INSTANCE TYPE", 0},
		{"ZONE", 0},
		{"CAPACITY", 0},
		{"NODEPOOL", 1},
	}
	for i, c := range cols {
		cell := tview.NewTableCell(c.title).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(c.exp)
		v.table.SetCell(0, i, cell)
	}
}

func (v *NodesView) renderRows() {
	for v.table.GetRowCount() > 1 {
		v.table.RemoveRow(1)
	}
	for i, n := range v.filtered {
		row := i + 1
		color := tcell.ColorWhite
		prefix := "  "
		if v.app.selected[n.Name] {
			color = tcell.ColorGreen
			prefix = "✓ "
		}

		nameCell := tview.NewTableCell(prefix+n.Name).
			SetTextColor(color).
			SetExpansion(3).
			SetReference(n.Name)

		v.table.SetCell(row, 0, nameCell)
		v.table.SetCell(row, 1, tview.NewTableCell(n.Status).SetTextColor(nodeStatusColor(n.Status)))
		v.table.SetCell(row, 2, tview.NewTableCell(n.Labels["beta.kubernetes.io/instance-type"]).SetTextColor(color))
		v.table.SetCell(row, 3, tview.NewTableCell(n.Labels["topology.kubernetes.io/zone"]).SetTextColor(color))
		v.table.SetCell(row, 4, tview.NewTableCell(n.Labels["karpenter.sh/capacity-type"]).SetTextColor(color))
		v.table.SetCell(row, 5, tview.NewTableCell(n.Labels["karpenter.sh/nodepool"]).SetTextColor(color).SetExpansion(1))
	}
}

func (v *NodesView) showTopStats() {
	row, _ := v.table.GetSelection()
	if row < 1 || row > len(v.filtered) {
		v.app.setStatus("[red]Select a node first[-]")
		return
	}
	ref := v.table.GetCell(row, 0).GetReference()
	if ref == nil {
		return
	}
	name := ref.(string)

	v.app.setStatus(fmt.Sprintf("[yellow]kubectl top node %s...[-]", name))
	out, err := kubectl.TopNode(name)
	if err != nil {
		v.app.setStatus(fmt.Sprintf("[red]%v[-]", err))
		return
	}

	v.app.tv.QueueUpdateDraw(func() {
		content := fmt.Sprintf("[yellow]kubectl top node %s[-]\n\n%s\n\n[gray]Press Esc or q to close[-]", name, out)
		modal := newTextModal(v.app, content, "top-modal")
		v.app.pages.AddPage("top-modal", modal, true, true)
		v.app.tv.SetFocus(modal)
	})
}

func nodeStatusColor(status string) tcell.Color {
	switch status {
	case "Ready":
		return tcell.ColorGreen
	case "NotReady":
		return tcell.ColorRed
	}
	return tcell.ColorYellow
}
