# ssm-me

A terminal UI (TUI) for managing AWS SSM commands against Kubernetes nodes.

## Features

- **Nodes view** — lists nodes from the current `kubectl` context with their labels (instance type, zone, capacity type, nodepool)
- **Label filtering** — filter nodes by `key=value` pairs, or by a bare `key` substring to match any label whose *key* contains it (e.g. `topology.gemini.com` matches every `topology.gemini.com/*` label)
- **Matched labels column** — when a filter is active, the specific label(s) that matched are shown per row
- **Multi-select** — select one or more nodes for targeted SSM execution
- **Interactive SSM session** — press `s` on a node to open a live `aws ssm start-session` shell against it, right from the TUI
- **Execute view** — send shell commands via AWS SSM (`AWS-RunShellScript`) to selected nodes
- **Execution history** — all runs are stored in `/tmp/ssm-me/`; view stdout/stderr per execution
- **Node stats** — `kubectl top node` output shown inline

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
| `Space` / `Enter` | Select/deselect node (nodes view) |
| `e` | Go to Execute with current selection |
| `s` | Open an interactive SSM session against the highlighted node |
| `r` | Refresh nodes / execution history |
| `f` or `/` | Focus filter input |
| `t` | Show `kubectl top node` stats for selected row |
| `Esc` | Clear selection (nodes) / back to nodes (execute) |
| `Ctrl+E` | Send SSM command |
| `d` | Delete execution from history |
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

Executions are persisted in `/tmp/ssm-me/`:

- `executions.json` — index of all runs (command ID, nodes, status, timestamp)
- `<uuid>.txt` — stdout/stderr output per execution
