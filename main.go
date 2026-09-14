package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Alagroc/ssm-me/pkg/awsclient"
	"github.com/Alagroc/ssm-me/pkg/debuglog"
	"github.com/Alagroc/ssm-me/pkg/store"
	"github.com/Alagroc/ssm-me/pkg/ui"
)

const helpText = `ssm-me — terminal UI for AWS SSM on Kubernetes nodes

USAGE
  ssm-me [--help] [--debug-log]

KEYBINDINGS

  Navigation
    1              Nodes view
    2              Execute view
    3              History view
    4              Settings view
    Left / Right   Cycle tabs
    Shift+E        Jump to History (execution results)
    Q              Quit

  Nodes view
    r              Refresh node list from kubectl
    /              Focus filter input
    Space / Enter  Select / deselect node
    e              Go to Execute with current selection
    s              Open interactive SSM session for highlighted node
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
    Shift+D        Delete ALL executions (confirms first)
    Esc / q        Close output modal

  Settings view
    Tab / Down     Next field
    Enter          Change color scheme / toggle debug log

FILTER SYNTAX
  Comma-separated key=value pairs, or a bare key substring to match any
  label whose key contains it:
    karpenter.sh/capacity-type=spot
    karpenter.sh/capacity-type=on-demand,topology.kubernetes.io/zone=us-east-1a
    topology.gemini.com

STORAGE
  Executions are stored in /tmp/ssm-me/
    executions.json   index of all runs
    <uuid>.txt        stdout/stderr per execution

DEBUG LOG
  --debug-log (or "Enable debug log" in the Settings view) writes a full
  trace of every aws/kubectl command (args, stdout, stderr, error) and
  status-bar message to /tmp/<random>-ssm-me.log (path printed to stderr
  on startup, or shown in the status bar when enabled from Settings).

SETTINGS
  Color scheme, debug log, and auto-refresh-on-startup preferences are
  persisted to ~/.ssm-me/settings.json (survives reboots, unlike the
  /tmp execution history) and reloaded on the next run.
`

func main() {
	help := flag.Bool("help", false, "show keybindings and usage")
	flag.BoolVar(help, "h", false, "show keybindings and usage")
	debugLog := flag.Bool("debug-log", false, "write a full command/status trace to /tmp/<random>-ssm-me.log")
	flag.Parse()

	if *help {
		fmt.Fprint(os.Stdout, helpText)
		os.Exit(0)
	}

	if err := store.Init(); err != nil {
		log.Fatalf("init store: %v", err)
	}

	settings, err := store.LoadSettings()
	if err != nil {
		log.Printf("load settings: %v (using defaults)", err)
		settings = store.Settings{Theme: "default"}
	}

	if *debugLog || settings.DebugLog {
		if logPath, err := debuglog.SetEnabled(true); err != nil {
			log.Printf("debug log: %v (continuing without it)", err)
		} else {
			fmt.Fprintf(os.Stderr, "ssm-me: debug log at %s\n", logPath)
			debuglog.Printf("ssm-me starting")
		}
	}

	awsClient, err := awsclient.New()
	if err != nil {
		log.Printf("aws: %v — SSM features disabled until this is fixed", err)
	}

	app := ui.NewApp(awsClient, settings)
	if err := app.Run(); err != nil {
		log.Fatalf("run: %v", err)
	}
}
