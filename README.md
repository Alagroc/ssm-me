# ssm-me

A terminal UI (TUI) for managing AWS SSM commands against Kubernetes nodes.

## Features

- **Nodes view** — lists nodes from the current `kubectl` context with their labels (instance type, zone, capacity type, nodepool)
- **Label filtering** — filter nodes by one or more `key=value` label pairs
- **Multi-select** — select one or more nodes for targeted SSM execution
- **Execute view** — send shell commands via AWS SSM (`AWS-RunShellScript`) to selected nodes
- **Execution history** — all runs are stored in `/tmp/ssm-me/`; view stdout/stderr per execution
- **Node stats** — `kubectl top node` output shown inline

## Requirements

- `kubectl` configured with a valid context
- AWS credentials (env vars, `~/.aws/credentials`, or instance role) with permissions for:
  - `ec2:DescribeInstances`
  - `ssm:SendCommand`
  - `ssm:GetCommandInvocation`
- SSM agent running on target EC2 nodes

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
| `Space` / `Enter` | Select/deselect node (nodes view) |
| `e` | Go to Execute with current selection |
| `r` | Refresh nodes / execution history |
| `f` or `/` | Focus filter input |
| `t` | Show `kubectl top node` stats for selected row |
| `Esc` | Clear selection (nodes) / back to nodes (execute) |
| `Ctrl+E` | Send SSM command |
| `d` | Delete execution from history |
| `q` / `Esc` | Close modal |
| `Q` | Quit |

### Filtering nodes

In the filter box, enter comma-separated `key=value` pairs:

```
karpenter.sh/capacity-type=spot
karpenter.sh/capacity-type=on-demand,topology.kubernetes.io/zone=us-east-1a
```

## Storage

Executions are persisted in `/tmp/ssm-me/`:

- `executions.json` — index of all runs (command ID, nodes, status, timestamp)
- `<uuid>.txt` — stdout/stderr output per execution
