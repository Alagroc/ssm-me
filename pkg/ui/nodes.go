package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Alagroc/ssm-me/pkg/awsclient"
	"github.com/Alagroc/ssm-me/pkg/kubectl"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// sortColumns maps a sort shortcut key (rune) to the column it sorts, its
// display label (for the status message), and the table column index whose
// header gets the ▲/▼ indicator.
var sortColumns = []struct {
	key      rune
	field    string
	label    string
	colIndex int
}{
	{'S', "status", "Status", 1},
	{'I', "instance-type", "Instance Type", 2},
	{'C', "capacity", "Capacity", 4},
	{'N', "nodepool", "Nodepool", 5},
	{'L', "labels", "Matched Labels", 6},
}

type NodesView struct {
	app           *App
	root          *tview.Flex
	table         *tview.Table
	filter        *tview.InputField
	info          *tview.TextView
	help          *tview.TextView
	filtered      []kubectl.Node
	activeFilters []kubectl.Filter

	sortField string // one of sortColumns[i].field, or "" for unsorted
	sortDir   int    // 1 = ascending, -1 = descending (meaningless when sortField == "")
}

func newNodesView(app *App) *NodesView {
	v := &NodesView{app: app}

	v.filter = tview.NewInputField().
		SetLabel("Filter (key=val,...): ").
		SetFieldWidth(50)
	v.filter.SetDoneFunc(func(_ tcell.Key) {
		v.applyFilter()
		v.renderFiltered()
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
		app.renderHeader(pageNodes)
	})

	v.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune {
			for _, sc := range sortColumns {
				if event.Rune() == sc.key {
					v.toggleSort(sc.field, sc.label)
					return nil
				}
			}
		}
		switch {
		case event.Key() == tcell.KeyRune && event.Rune() == 'r':
			go v.refresh()
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == '/':
			app.tv.SetFocus(v.filter)
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == 'e':
			app.switchTo(pageExecute)
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == 't':
			go v.showTopStats()
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == 's':
			go v.startSession()
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == 'c':
			go v.pickContext()
			return nil
		case event.Key() == tcell.KeyEsc:
			app.selected = make(map[string]bool)
			v.renderRows()
			app.renderHeader(pageNodes)
			return nil
		}
		return event
	})

	v.info = tview.NewTextView().SetDynamicColors(true)

	v.help = tview.NewTextView().SetDynamicColors(true)
	v.updateHelp()

	v.root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.filter, 1, 0, false).
		AddItem(v.table, 0, 1, true).
		AddItem(v.info, 1, 0, false).
		AddItem(v.help, 1, 0, false)

	v.renderHeader()
	return v
}

func (v *NodesView) updateHelp() {
	v.help.SetText(" " + accentTag("Space/Enter") + ":select  " + accentTag("e") + ":execute  " +
		accentTag("s") + ":ssm session  " + accentTag("c") + ":context  " + accentTag("r") + ":refresh  " + accentTag("/") + ":filter  " +
		accentTag("t") + ":top  " + accentTag("S/I/C/N/L") + ":sort  " + accentTag("Esc") + ":clear selection  " +
		accentTag("◄►") + "/" + accentTag("1-4") + ":tabs  " + accentTag("Shift+E") + ":results  " + accentTag("Q") + ":quit")
}

// applyTheme re-colors this view's primitives and re-renders its
// theme-tagged text after a live theme switch. Must run on the main
// event-loop goroutine (see App.applyThemeLive).
func (v *NodesView) applyTheme() {
	v.filter.SetBackgroundColor(activeTheme.Background)
	v.table.SetBackgroundColor(activeTheme.Background)
	v.info.SetBackgroundColor(activeTheme.Background)
	v.help.SetBackgroundColor(activeTheme.Background)

	v.renderHeader()
	v.renderRows()
	v.updateHelp()
	v.info.SetText(fmt.Sprintf(" %s", infoTag(fmt.Sprintf("%d/%d nodes", len(v.filtered), len(v.app.nodes)))))
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
	v.app.tv.QueueUpdateDraw(v.renderFiltered)
	v.app.setStatus(fmt.Sprintf("[green]%d nodes[-]", len(nodes)))
}

func (v *NodesView) applyFilter() {
	v.activeFilters = kubectl.ParseFilters(v.filter.GetText())
	v.filtered = kubectl.FilterNodes(v.app.nodes, v.activeFilters)
}

func (v *NodesView) renderFiltered() {
	v.renderRows()
	v.info.SetText(" " + infoTag(fmt.Sprintf("%d/%d nodes", len(v.filtered), len(v.app.nodes))))
}

// toggleSort cycles the given column through ascending -> descending ->
// unsorted. Called synchronously from the table's input capture (main
// event-loop goroutine), so it updates primitives directly.
func (v *NodesView) toggleSort(field, label string) {
	switch {
	case v.sortField != field:
		v.sortField, v.sortDir = field, 1
	case v.sortDir == 1:
		v.sortDir = -1
	default:
		v.sortField, v.sortDir = "", 0
	}

	v.renderHeader()
	v.renderRows()

	switch v.sortDir {
	case 1:
		v.app.status.SetText(" " + infoTag(fmt.Sprintf("Sorted by %s (ascending)", label)))
	case -1:
		v.app.status.SetText(" " + infoTag(fmt.Sprintf("Sorted by %s (descending)", label)))
	default:
		v.app.status.SetText(" " + infoTag("Sort cleared"))
	}
}

// sortKey returns the value n is compared on for the current sort field.
func (v *NodesView) sortKey(n kubectl.Node) string {
	switch v.sortField {
	case "status":
		return n.Status
	case "instance-type":
		return n.Labels["beta.kubernetes.io/instance-type"]
	case "capacity":
		return n.Labels["karpenter.sh/capacity-type"]
	case "nodepool":
		return n.Labels["karpenter.sh/nodepool"]
	case "labels":
		return strings.Join(kubectl.MatchedLabels(n, v.activeFilters), ", ")
	}
	return n.Name
}

// sortedNodes returns v.filtered in display order, sorted by the active
// sort field if any — a copy, so v.filtered's own order (kubectl's
// original order) is never disturbed.
func (v *NodesView) sortedNodes() []kubectl.Node {
	if v.sortField == "" {
		return v.filtered
	}
	sorted := make([]kubectl.Node, len(v.filtered))
	copy(sorted, v.filtered)
	sort.SliceStable(sorted, func(i, j int) bool {
		ki, kj := v.sortKey(sorted[i]), v.sortKey(sorted[j])
		if v.sortDir < 0 {
			return ki > kj
		}
		return ki < kj
	})
	return sorted
}

// nodeAt returns the node backing the table row at the given index (1-based,
// row 0 is the header), looked up by its NAME reference rather than by
// position — sortedNodes() can render rows in a different order than
// v.filtered, so a positional index into v.filtered would pick the wrong
// node once sorting is active.
func (v *NodesView) nodeAt(row int) (kubectl.Node, bool) {
	ref := v.table.GetCell(row, 0).GetReference()
	if ref == nil {
		return kubectl.Node{}, false
	}
	name := ref.(string)
	for _, n := range v.filtered {
		if n.Name == name {
			return n, true
		}
	}
	return kubectl.Node{}, false
}

func (v *NodesView) renderHeader() {
	cols := []struct {
		title string
		exp   int
		field string
	}{
		{"NAME", 3, ""},
		{"STATUS", 0, "status"},
		{"INSTANCE TYPE", 0, "instance-type"},
		{"ZONE", 0, ""},
		{"CAPACITY", 0, "capacity"},
		{"NODEPOOL", 1, "nodepool"},
		{"MATCHED LABELS", 3, "labels"},
	}
	for i, c := range cols {
		title := c.title
		if c.field != "" && c.field == v.sortField {
			if v.sortDir < 0 {
				title += " ▼"
			} else {
				title += " ▲"
			}
		}
		cell := tview.NewTableCell(title).
			SetTextColor(activeTheme.Accent).
			SetSelectable(false).
			SetExpansion(c.exp)
		v.table.SetCell(0, i, cell)
	}
}

func (v *NodesView) renderRows() {
	for v.table.GetRowCount() > 1 {
		v.table.RemoveRow(1)
	}
	for i, n := range v.sortedNodes() {
		row := i + 1
		color := tcell.ColorWhite
		prefix := "  "
		if v.app.selected[n.Name] {
			color = activeTheme.Accent
			prefix = "✓ "
		}

		nameCell := tview.NewTableCell(prefix + n.Name).
			SetTextColor(color).
			SetExpansion(3).
			SetReference(n.Name)

		v.table.SetCell(row, 0, nameCell)
		v.table.SetCell(row, 1, tview.NewTableCell(n.Status).SetTextColor(nodeStatusColor(n.Status)))
		v.table.SetCell(row, 2, tview.NewTableCell(n.Labels["beta.kubernetes.io/instance-type"]).SetTextColor(color))
		v.table.SetCell(row, 3, tview.NewTableCell(n.Labels["topology.kubernetes.io/zone"]).SetTextColor(color))
		v.table.SetCell(row, 4, tview.NewTableCell(n.Labels["karpenter.sh/capacity-type"]).SetTextColor(color))
		v.table.SetCell(row, 5, tview.NewTableCell(n.Labels["karpenter.sh/nodepool"]).SetTextColor(color).SetExpansion(1))

		matched := strings.Join(kubectl.MatchedLabels(n, v.activeFilters), ", ")
		v.table.SetCell(row, 6, tview.NewTableCell(matched).SetTextColor(activeTheme.Info).SetExpansion(3))
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
		content := fmt.Sprintf("%s\n\n%s\n\n%s", accentTag("kubectl top node "+name), out, infoTag("Press Esc or q to close"))
		modal := newTextModal(v.app, content, "top-modal")
		v.app.pages.AddPage("top-modal", modal, true, true)
		v.app.tv.SetFocus(modal)
	})
}

// startSession resolves the highlighted node's instance ID (preferring the
// kubectl.InstanceIDLabel, falling back to an EC2 lookup) and launches an
// interactive `aws ssm start-session` against it, suspending the TUI's
// screen so the session gets the real terminal.
func (v *NodesView) startSession() {
	row, _ := v.table.GetSelection()
	n, ok := v.nodeAt(row)
	if !ok {
		v.app.setStatus("[red]Select a node first[-]")
		return
	}
	if v.app.aws == nil {
		v.app.setStatus("[red]aws CLI not found — install it and restart[-]")
		return
	}

	id, ok := n.Labels[kubectl.InstanceIDLabel]
	if !ok || id == "" {
		v.app.setStatus(fmt.Sprintf("[yellow]Resolving instance ID for %s...[-]", n.Name))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		resolved, err := v.app.aws.ResolveInstanceIDs(ctx, []string{n.Name})
		cancel()
		if err != nil {
			v.app.setStatus(fmt.Sprintf("[red]EC2 lookup failed: %v[-]", err))
			return
		}
		id, ok = resolved[n.Name]
		if !ok || id == "" {
			v.app.setStatus(fmt.Sprintf("[red]Could not resolve instance ID for %s[-]", n.Name))
			return
		}
	}

	v.app.setStatus(fmt.Sprintf("[yellow]Starting SSM session to %s (%s)...[-]", n.Name, id))
	resumed := v.app.tv.Suspend(func() {
		if err := awsclient.StartSession(id); err != nil {
			fmt.Printf("\nssm-me: session ended: %v\n", err)
		}
	})
	if !resumed {
		v.app.setStatus("[red]Could not suspend terminal for SSM session[-]")
		return
	}
	v.app.setStatus(fmt.Sprintf("[green]Session with %s closed[-]", n.Name))
}

// pickContext lists the contexts in the local kubeconfig and shows a
// picker to switch between them without restarting the app.
func (v *NodesView) pickContext() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	v.app.setStatus("[yellow]Loading kubectl contexts...[-]")
	contexts, err := kubectl.ListContexts(ctx)
	if err != nil {
		v.app.setStatus(fmt.Sprintf("[red]%v[-]", err))
		return
	}
	if len(contexts) == 0 {
		v.app.setStatus("[red]No kubectl contexts found[-]")
		return
	}

	v.app.setStatus(infoTag("Select a context — Esc to cancel"))
	v.app.tv.QueueUpdateDraw(func() {
		v.showContextPicker(contexts)
	})
}

func (v *NodesView) showContextPicker(contexts []string) {
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(true)
	list.SetTitle(" Select kubectl context (Esc to cancel) ")

	current := 0
	for i, c := range contexts {
		label := c
		if c == v.app.context {
			label += "  (current)"
			current = i
		}
		list.AddItem(label, "", 0, nil)
	}
	list.SetCurrentItem(current)

	closePicker := func() {
		v.app.pages.RemovePage("context-picker")
		v.app.tv.SetFocus(v.table)
	}
	list.SetDoneFunc(closePicker)
	list.SetSelectedFunc(func(index int, _, _ string, _ rune) {
		name := contexts[index]
		closePicker()
		if name != v.app.context {
			go v.applyContext(name)
		}
	})

	v.app.pages.AddPage("context-picker", list, true, true)
	v.app.tv.SetFocus(list)
}

func (v *NodesView) applyContext(name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	v.app.setStatus(fmt.Sprintf("[yellow]Switching to context %s...[-]", name))
	if err := kubectl.UseContext(ctx, name); err != nil {
		v.app.setStatus(fmt.Sprintf("[red]%v[-]", err))
		return
	}

	v.app.context = name
	v.app.tv.QueueUpdateDraw(func() { v.app.renderHeader(pageNodes) })
	v.refresh()
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
