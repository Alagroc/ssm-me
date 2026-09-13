package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Alagroc/ssm-me/pkg/store"
	"github.com/Alagroc/ssm-me/pkg/ui"
)

const helpText = `ssm-me — terminal UI for AWS SSM on Kubernetes nodes

USAGE
  ssm-me [--help]

KEYBINDINGS

  Navigation
    1              Nodes view
    2              Execute view
    3              History view
    Q              Quit

  Nodes view
    r              Refresh node list from kubectl
    f  /           Focus filter input
    Space / Enter  Select / deselect node
    e              Go to Execute with current selection
    t              kubectl top node for highlighted row
    Esc            Clear selection

  Execute view
    Ctrl+E         Send SSM command to selected nodes
    Tab            Cycle between fields
    Esc            Back to Nodes

  History view
    Enter          View command output
    r              Refresh execution list
    d              Delete selected execution
    Esc / q        Close output modal

FILTER SYNTAX
  Comma-separated key=value label pairs:
    karpenter.sh/capacity-type=spot
    karpenter.sh/capacity-type=on-demand,topology.kubernetes.io/zone=us-east-1a

STORAGE
  Executions are stored in /tmp/ssm-me/
    executions.json   index of all runs
    <uuid>.txt        stdout/stderr per execution
`

func main() {
	help := flag.Bool("help", false, "show keybindings and usage")
	flag.BoolVar(help, "h", false, "show keybindings and usage")
	flag.Parse()

	if *help {
		fmt.Fprint(os.Stdout, helpText)
		os.Exit(0)
	}

	if err := store.Init(); err != nil {
		log.Fatalf("init store: %v", err)
	}
	app := ui.NewApp()
	if err := app.Run(); err != nil {
		log.Fatalf("run: %v", err)
	}
}
