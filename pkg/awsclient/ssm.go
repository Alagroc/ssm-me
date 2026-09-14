package awsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Alagroc/ssm-me/pkg/debuglog"
)

type CommandResult struct {
	CommandID string
	Status    string
}

type ssmTarget struct {
	Key    string   `json:"Key"`
	Values []string `json:"Values"`
}

type sendCommandOutput struct {
	Command struct {
		CommandId string `json:"CommandId"`
		Status    string `json:"Status"`
	} `json:"Command"`
}

// SendCommand runs `aws ssm send-command` with the AWS-RunShellScript
// document against the given instance IDs.
func (c *Client) SendCommand(ctx context.Context, instanceIDs []string, command, comment string) (*CommandResult, error) {
	if comment == "" {
		comment = "ssm-me"
	}
	targets, err := json.Marshal([]ssmTarget{{Key: "instanceIds", Values: instanceIDs}})
	if err != nil {
		return nil, err
	}
	params, err := json.Marshal(map[string][]string{"commands": {command}})
	if err != nil {
		return nil, err
	}

	var out sendCommandOutput
	if err := runAWS(ctx, &out,
		"ssm", "send-command",
		"--document-name", "AWS-RunShellScript",
		"--comment", comment,
		"--targets", string(targets),
		"--parameters", string(params),
	); err != nil {
		return nil, fmt.Errorf("send command: %w", err)
	}
	return &CommandResult{CommandID: out.Command.CommandId, Status: out.Command.Status}, nil
}

type listInvocationsOutput struct {
	CommandInvocations []struct {
		InstanceId     string `json:"InstanceId"`
		Status         string `json:"Status"`
		StatusDetails  string `json:"StatusDetails"`
		CommandPlugins []struct {
			Output string `json:"Output"`
		} `json:"CommandPlugins"`
	} `json:"CommandInvocations"`
}

func listInvocations(ctx context.Context, commandID string) (listInvocationsOutput, error) {
	var out listInvocationsOutput
	err := runAWS(ctx, &out, "ssm", "list-command-invocations", "--command-id", commandID, "--details")
	return out, err
}

func isTerminalStatus(s string) bool {
	switch s {
	case "Success", "Failed", "TimedOut", "Cancelled", "Undeliverable", "Terminated":
		return true
	}
	return false
}

// PollUntilDone polls command status via list-command-invocations until all
// instances finish or timeout elapses. Returns a map of instanceID -> final
// status string.
func (c *Client) PollUntilDone(ctx context.Context, commandID string, instanceIDs []string, timeout time.Duration) (map[string]string, error) {
	deadline := time.Now().Add(timeout)
	statuses := make(map[string]string, len(instanceIDs))

	for time.Now().Before(deadline) {
		out, err := listInvocations(ctx, commandID)
		if err != nil {
			return statuses, err
		}

		done := 0
		seen := make(map[string]bool, len(out.CommandInvocations))
		for _, inv := range out.CommandInvocations {
			seen[inv.InstanceId] = true
			statuses[inv.InstanceId] = inv.Status
			if isTerminalStatus(inv.Status) {
				done++
			}
		}
		if done == len(instanceIDs) && len(seen) == len(instanceIDs) {
			break
		}

		select {
		case <-ctx.Done():
			return statuses, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return statuses, nil
}

// CollectOutput fetches command output for all instances via
// list-command-invocations and formats it.
func (c *Client) CollectOutput(ctx context.Context, commandID string, instanceIDs []string) string {
	out, err := listInvocations(ctx, commandID)
	if err != nil {
		return fmt.Sprintf("Error fetching output: %v\n", err)
	}

	var sb strings.Builder
	for _, inv := range out.CommandInvocations {
		fmt.Fprintf(&sb, "=== %s (%s) ===\n", inv.InstanceId, inv.Status)
		for _, plugin := range inv.CommandPlugins {
			if plugin.Output != "" {
				sb.WriteString(plugin.Output)
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// OverallStatus returns the worst-case status across all instances.
func OverallStatus(statuses map[string]string) string {
	for _, s := range statuses {
		if s != "Success" {
			return s
		}
	}
	return "Success"
}

// StartSession launches an interactive `aws ssm start-session` against the
// given instance, with the terminal's stdio wired through directly. It
// requires the session-manager-plugin to be installed alongside the AWS
// CLI. Callers must suspend the TUI's screen before invoking this.
func StartSession(instanceID string) error {
	debuglog.Printf("aws ssm start-session --target %s", instanceID)
	cmd := exec.Command("aws", "ssm", "start-session", "--target", instanceID)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	debuglog.Printf("aws ssm start-session --target %s: err: %v", instanceID, err)
	return err
}
