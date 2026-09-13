package awsclient

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type CommandResult struct {
	CommandID string
	Status    string
}

func (c *Client) SendCommand(ctx context.Context, instanceIDs []string, command, comment string) (*CommandResult, error) {
	if comment == "" {
		comment = "ssm-me"
	}
	out, err := c.SSM.SendCommand(ctx, &ssm.SendCommandInput{
		InstanceIds:  instanceIDs,
		DocumentName: aws.String("AWS-RunShellScript"),
		Parameters:   map[string][]string{"commands": {command}},
		Comment:      aws.String(comment),
	})
	if err != nil {
		return nil, fmt.Errorf("send command: %w", err)
	}
	return &CommandResult{
		CommandID: aws.ToString(out.Command.CommandId),
		Status:    string(out.Command.Status),
	}, nil
}

func (c *Client) GetInvocationOutput(ctx context.Context, commandID, instanceID string) (stdout, stderr string, err error) {
	out, err := c.SSM.GetCommandInvocation(ctx, &ssm.GetCommandInvocationInput{
		CommandId:  aws.String(commandID),
		InstanceId: aws.String(instanceID),
	})
	if err != nil {
		return "", "", fmt.Errorf("get invocation: %w", err)
	}
	return aws.ToString(out.StandardOutputContent), aws.ToString(out.StandardErrorContent), nil
}

// PollUntilDone polls command status until all instances finish or timeout elapses.
// Returns a map of instanceID -> final status string.
func (c *Client) PollUntilDone(ctx context.Context, commandID string, instanceIDs []string, timeout time.Duration) (map[string]string, error) {
	deadline := time.Now().Add(timeout)
	statuses := make(map[string]string, len(instanceIDs))

	for time.Now().Before(deadline) {
		done := 0
		for _, id := range instanceIDs {
			out, err := c.SSM.GetCommandInvocation(ctx, &ssm.GetCommandInvocationInput{
				CommandId:  aws.String(commandID),
				InstanceId: aws.String(id),
			})
			if err != nil {
				statuses[id] = "Error"
				done++
				continue
			}
			statuses[id] = aws.ToString(out.StatusDetails)
			if isTerminal(out.Status) {
				done++
			}
		}
		if done == len(instanceIDs) {
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

func isTerminal(s types.CommandInvocationStatus) bool {
	switch s {
	case types.CommandInvocationStatusSuccess,
		types.CommandInvocationStatusFailed,
		types.CommandInvocationStatusTimedOut,
		types.CommandInvocationStatusCancelled:
		return true
	}
	return false
}

// CollectOutput fetches stdout/stderr from all instances and formats them.
func (c *Client) CollectOutput(ctx context.Context, commandID string, instanceIDs []string) string {
	var sb strings.Builder
	for _, id := range instanceIDs {
		fmt.Fprintf(&sb, "=== %s ===\n", id)
		stdout, stderr, err := c.GetInvocationOutput(ctx, commandID, id)
		if err != nil {
			fmt.Fprintf(&sb, "Error: %v\n\n", err)
			continue
		}
		if stdout != "" {
			fmt.Fprintf(&sb, "STDOUT:\n%s\n", stdout)
		}
		if stderr != "" {
			fmt.Fprintf(&sb, "STDERR:\n%s\n", stderr)
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
