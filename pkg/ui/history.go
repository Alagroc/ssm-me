package ui

import (
	"fmt"
	"strings"

	"github.com/Alagroc/ssm-me/pkg/store"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type HistoryView struct {
	app   *App
	root  *tview.Flex
	table *tview.Table
	execs []store.Execution
}

func newHistoryView(app *App) *HistoryView {
	v := &HistoryView{app: app}

	v.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)

	v.table.SetSelectedFunc(func(row, _ int) {
		if row < 1 || row > len(v.execs) {
			return
		}
		go v.showOutput(v.execs[row-1])
	})

	v.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'r':
				v.refresh()
				return nil
			case 'd':
				v.deleteSelected()
				return nil
			}
		}
		return event
	})

	help := tview.NewTextView().
		SetDynamicColors(true).
		SetText(" [yellow]Enter[-]:view output  [yellow]r[-]:refresh  [yellow]d[-]:delete  [yellow]1-3[-]:tabs")

	v.root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.table, 0, 1, true).
		AddItem(help, 1, 0, false)

	v.renderHeader()
	return v
}

func (v *HistoryView) renderHeader() {
	headers := []struct {
		title string
		exp   int
	}{
		{"TIME", 0},
		{"COMMAND ID", 0},
		{"COMMAND", 2},
		{"NODES", 1},
		{"STATUS", 0},
	}
	for i, h := range headers {
		v.table.SetCell(0, i, tview.NewTableCell(h.title).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(h.exp))
	}
}

func (v *HistoryView) refresh() {
	execs, err := store.Load()
	if err != nil {
		v.app.setStatus(fmt.Sprintf("[red]Load history: %v[-]", err))
		return
	}
	// Newest first
	for i, j := 0, len(execs)-1; i < j; i, j = i+1, j-1 {
		execs[i], execs[j] = execs[j], execs[i]
	}
	v.execs = execs
	v.app.tv.QueueUpdateDraw(v.renderRows)
}

func (v *HistoryView) renderRows() {
	for v.table.GetRowCount() > 1 {
		v.table.RemoveRow(1)
	}
	for i, e := range v.execs {
		row := i + 1
		ts := e.Timestamp.Format("01-02 15:04:05")

		cmdID := e.CommandID
		if len(cmdID) > 14 {
			cmdID = cmdID[:14] + "…"
		}
		cmd := e.Command
		if len(cmd) > 50 {
			cmd = cmd[:50] + "…"
		}
		nodes := strings.Join(e.NodeNames, ", ")
		if len(nodes) > 35 {
			nodes = nodes[:35] + "…"
		}

		statusColor := executionStatusColor(e.Status)

		v.table.SetCell(row, 0, tview.NewTableCell(ts).SetTextColor(tcell.ColorGray).SetSelectable(true))
		v.table.SetCell(row, 1, tview.NewTableCell(cmdID).SetTextColor(tcell.ColorAqua).SetSelectable(true))
		v.table.SetCell(row, 2, tview.NewTableCell(cmd).SetTextColor(tcell.ColorWhite).SetExpansion(2).SetSelectable(true))
		v.table.SetCell(row, 3, tview.NewTableCell(nodes).SetTextColor(tcell.ColorGray).SetExpansion(1).SetSelectable(true))
		v.table.SetCell(row, 4, tview.NewTableCell(e.Status).SetTextColor(statusColor).SetSelectable(true))
	}
	v.app.status.SetText(fmt.Sprintf(" [gray]%d executions — press Enter to view output[-]", len(v.execs)))
}

func (v *HistoryView) showOutput(exec store.Execution) {
	output, err := store.LoadOutput(exec.ID)
	if err != nil {
		output = fmt.Sprintf("[Output not yet available]\n\nCommand ID: %s\nStatus: %s\n\nOutput is saved once the command completes.", exec.CommandID, exec.Status)
	}

	header := fmt.Sprintf("[yellow]Command:[-]  %s\n[yellow]ID:[-]       %s\n[yellow]Nodes:[-]    %s\n[yellow]Time:[-]     %s\n[yellow]Status:[-]   %s\n\n",
		exec.Command,
		exec.CommandID,
		strings.Join(exec.NodeNames, ", "),
		exec.Timestamp.Format("2006-01-02 15:04:05"),
		exec.Status,
	)

	v.app.tv.QueueUpdateDraw(func() {
		modal := newTextModal(v.app, header+output, "output-modal")
		modal.SetTitle(fmt.Sprintf(" Output: %s ", truncate(exec.CommandID, 14)))
		v.app.pages.AddPage("output-modal", modal, true, true)
		v.app.tv.SetFocus(modal)
	})
}

func (v *HistoryView) deleteSelected() {
	row, _ := v.table.GetSelection()
	if row < 1 || row > len(v.execs) {
		return
	}
	exec := v.execs[row-1]
	if err := store.Delete(exec.ID); err != nil {
		v.app.setStatus(fmt.Sprintf("[red]Delete failed: %v[-]", err))
		return
	}
	v.app.setStatus(fmt.Sprintf("[green]Deleted %s[-]", truncate(exec.CommandID, 14)))
	v.refresh()
}

func executionStatusColor(status string) tcell.Color {
	switch status {
	case "Success":
		return tcell.ColorGreen
	case "Running", "Pending", "InProgress":
		return tcell.ColorYellow
	case "Failed", "TimedOut", "Cancelled", "DeliveryTimedOut", "ExecutionTimedOut", "Error":
		return tcell.ColorRed
	}
	return tcell.ColorWhite
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
