package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Alagroc/ssm-me/pkg/awsclient"
	"github.com/Alagroc/ssm-me/pkg/store"
	"github.com/gdamore/tcell/v2"
	"github.com/google/uuid"
	"github.com/rivo/tview"
)

type ExecuteView struct {
	app     *App
	root    *tview.Flex
	nodeBox *tview.TextView
	cmd     *tview.TextArea
	comment *tview.InputField
}

func newExecuteView(app *App) *ExecuteView {
	v := &ExecuteView{app: app}

	v.nodeBox = tview.NewTextView().SetDynamicColors(true)
	v.nodeBox.SetBorder(true)
	v.nodeBox.SetTitle(" Selected Nodes ")

	v.cmd = tview.NewTextArea().SetPlaceholder("Enter shell command...")
	v.cmd.SetBorder(true)
	v.cmd.SetTitle(" Command (Ctrl+E to execute) ")

	v.comment = tview.NewInputField().
		SetLabel("Comment (optional): ").
		SetFieldWidth(60).
		SetFieldTextColor(tcell.ColorWhite)

	help := tview.NewTextView().
		SetDynamicColors(true).
		SetText(" [yellow]Ctrl+E[-]:execute  [yellow]Tab[-]:next field  [yellow]Esc[-]:back to nodes  [yellow]1-3[-]:tabs")

	v.root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.nodeBox, 0, 1, false).
		AddItem(v.cmd, 0, 2, true).
		AddItem(v.comment, 1, 0, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(help, 1, 0, false)

	v.root.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlE {
			go v.execute()
			return nil
		}
		if event.Key() == tcell.KeyEsc {
			app.switchTo(pageNodes)
			return nil
		}
		return event
	})

	return v
}

func (v *ExecuteView) update() {
	var names []string
	for name, ok := range v.app.selected {
		if ok {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		v.nodeBox.SetText("[gray]No nodes selected. Go to Nodes (1) and press Space/Enter to select.[-]")
	} else {
		v.nodeBox.SetText("[green]" + strings.Join(names, "\n") + "[-]")
	}
}

func (v *ExecuteView) execute() {
	if v.app.aws == nil {
		v.app.setStatus("[red]AWS client not initialized — check credentials / region[-]")
		return
	}

	cmd := strings.TrimSpace(v.cmd.GetText())
	if cmd == "" {
		v.app.setStatus("[red]Command is empty[-]")
		return
	}

	var nodeNames []string
	for name, ok := range v.app.selected {
		if ok {
			nodeNames = append(nodeNames, name)
		}
	}
	if len(nodeNames) == 0 {
		v.app.setStatus("[red]No nodes selected[-]")
		return
	}

	ctx := context.Background()
	v.app.setStatus(fmt.Sprintf("[yellow]Resolving instance IDs for %d nodes...[-]", len(nodeNames)))

	instanceMap, err := v.app.aws.ResolveInstanceIDs(ctx, nodeNames)
	if err != nil {
		v.app.setStatus(fmt.Sprintf("[red]EC2 lookup failed: %v[-]", err))
		return
	}

	var instanceIDs, unresolved []string
	for _, name := range nodeNames {
		if id, ok := instanceMap[name]; ok {
			instanceIDs = append(instanceIDs, id)
		} else {
			unresolved = append(unresolved, name)
		}
	}
	if len(unresolved) > 0 {
		v.app.setStatus(fmt.Sprintf("[yellow]Warning: could not resolve: %s[-]", strings.Join(unresolved, ", ")))
	}
	if len(instanceIDs) == 0 {
		v.app.setStatus("[red]No instances resolved — check EC2 region and permissions[-]")
		return
	}

	comment := v.comment.GetText()
	v.app.setStatus(fmt.Sprintf("[yellow]Sending SSM command to %d instances...[-]", len(instanceIDs)))

	result, err := v.app.aws.SendCommand(ctx, instanceIDs, cmd, comment)
	if err != nil {
		v.app.setStatus(fmt.Sprintf("[red]SSM send failed: %v[-]", err))
		return
	}

	exec := store.Execution{
		ID:          uuid.New().String(),
		CommandID:   result.CommandID,
		NodeNames:   nodeNames,
		InstanceIDs: instanceIDs,
		Command:     cmd,
		Comment:     comment,
		Timestamp:   time.Now(),
		Status:      "Running",
	}
	if err := store.Save(exec); err != nil {
		v.app.setStatus(fmt.Sprintf("[red]Store error: %v[-]", err))
	}

	v.app.setStatus(fmt.Sprintf("[green]Sent! CommandID: %s — polling for completion[-]", result.CommandID))
	go v.pollCompletion(exec)
}

func (v *ExecuteView) pollCompletion(exec store.Execution) {
	ctx := context.Background()
	statuses, _ := v.app.aws.PollUntilDone(ctx, exec.CommandID, exec.InstanceIDs, 5*time.Minute)

	exec.Status = awsclient.OverallStatus(statuses)
	_ = store.Save(exec)

	output := v.app.aws.CollectOutput(ctx, exec.CommandID, exec.InstanceIDs)
	_ = store.SaveOutput(exec.ID, output)

	v.app.setStatus(fmt.Sprintf("[green]Command %s... finished: %s[-]", truncate(exec.CommandID, 8), exec.Status))
}
