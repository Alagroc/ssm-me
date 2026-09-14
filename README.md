# ssm-me

A terminal UI (TUI) for managing AWS SSM commands against Kubernetes nodes.

## Features

- **Nodes view** — lists nodes from the current `kubectl` context with their labels (instance type, zone, capacity type, nodepool)
- **Context switching** — press `c` to pick a different `kubectl` context from your kubeconfig and reload nodes against it, no restart needed
- **Label filtering** — filter nodes by `key=value` pairs, or by a bare `key` substring to match any label whose *key* contains it (e.g. `topology.gemini.com` matches every `topology.gemini.com/*` label)
- **Matched labels column** — when a filter is active, the specific label(s) that matched are shown per row
- **Multi-select** — select one or more nodes for targeted SSM execution
- **Interactive SSM session** — press `s` on a node to open a live `aws ssm start-session` shell against it, right from the TUI
- **Execute view** — send shell commands via AWS SSM (`AWS-RunShellScript`) to selected nodes
- **Execution history** — all runs are stored in `/tmp/ssm-me/`; view stdout/stderr per execution
- **Node stats** — `kubectl top node` output shown inline
- **Settings view** — switch color scheme (Default/Dark/High Contrast) and toggle the debug log live, no restart needed; choices persist to `~/.ssm-me/settings.json`

## Requirements

- `kubectl` configured with a valid context
- [AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) on `PATH`, configured with credentials (env vars, `~/.aws/credentials`, or instance role) — ssm-me shells out to `aws` for every SSM/EC2 operation rather than using the AWS SDK directly
- [Session Manager plugin](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html) on `PATH` — required by `aws ssm start-session` for interactive sessions
- IAM permissions for:
  - `ssm:StartSession` / `ssm:TerminateSession` (interactive sessions)
  - `ssm:SendCommand`, `ssm:ListCommandInvocations` (Execute view)
  - `ec2:DescribeInstances` (fallback instance ID lookup — see below)
- SSM agent running on target nodes

### Instance ID resolution

For both interactive sessions and Execute, ssm-me needs each node's EC2/managed-instance ID. It first checks the node's `topology.gemini.com/instance-id` label; if that label isn't present, it falls back to an `aws ec2 describe-instances` lookup by private DNS name.

## Install

```bash
go install github.com/Alagroc/ssm-me@latest
```

Or build from source:

```bash
git clone git@github.com:Alagroc/ssm-me.git
cd ssm-me
go build -o ssm-me .
```

## Usage

```bash
./ssm-me
```

### Key bindings

| Key | Action |
|-----|--------|
| `1` | Nodes view |
| `2` | Execute view |
| `3` | History view |
| `4` | Settings view |
| `Left` / `Right` | Cycle tabs |
| `Shift+E` | Jump to History (execution results) |
| `Space` / `Enter` | Select/deselect node (nodes view) |
| `e` | Go to Execute with current selection |
| `s` | Open an interactive SSM session against the highlighted node |
| `r` | Refresh nodes / execution history |
| `/` | Focus filter input |
| `t` | Show `kubectl top node` stats for selected row |
| `c` | Switch `kubectl` context |
| `Esc` | Clear selection (nodes) / back to nodes (execute) |
| `Ctrl+E` | Send SSM command |
| `d` | Delete execution from history |
| `Shift+D` | Delete ALL executions from history (confirms first) |
| `q` / `Esc` | Close modal |
| `Q` | Quit |

### Filtering nodes

In the filter box, enter comma-separated tokens. Each token is either an exact `key=value` label match, or a bare `key` substring that matches any label whose key contains it:

```
karpenter.sh/capacity-type=spot
karpenter.sh/capacity-type=on-demand,topology.kubernetes.io/zone=us-east-1a
topology.gemini.com
```

The last example matches every node carrying any `topology.gemini.com/*` label, and the matching label(s) are shown in the MATCHED LABELS column.

## Storage

Execution history is persisted in `/tmp/ssm-me/`:

- `executions.json` — index of all runs (command ID, nodes, status, timestamp)
- `<uuid>.txt` — stdout/stderr output per execution

Note: `/tmp` is typically cleared on reboot, so expect execution history to reset after a restart of the machine, not just the app.

## Settings

Open the Settings view (`4`) to change:

- **Color scheme** — Default (blue, classic ncurses "blue screen" look à la iptraf/Midnight Commander), Dark, or High Contrast. Applies immediately, no restart.
- **Enable debug log** — see below. Also applies immediately.
- **Auto-refresh nodes on startup** — skip pressing `r` after launch; the node list loads automatically.

Unlike execution history, these preferences are meant to survive a reboot, so they're persisted separately to `~/.ssm-me/settings.json` and reloaded on the next run.

## Debugging

Enable the debug log — via `--debug-log` at startup, or the "Enable debug log" checkbox in Settings — to write a log to `/tmp/<random>-ssm-me.log` (the path is printed to stderr on startup, or shown in the status bar when enabled from Settings). It records every `aws`/`kubectl` command run — full argument list, stdout, stderr, and error — plus every status-bar message, since the status bar is a single line and truncates long errors. Tail it in another terminal while reproducing an issue:

```bash
./ssm-me --debug-log
# in another terminal:
tail -f /tmp/*-ssm-me.log
```
