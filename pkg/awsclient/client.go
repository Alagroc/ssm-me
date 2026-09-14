package awsclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Client shells out to the AWS CLI (v2) for all SSM/EC2 operations, and to
// the Session Manager plugin (via `aws ssm start-session`) for interactive
// sessions — there is no SDK equivalent for the interactive session's
// websocket streaming, so the CLI is the source of truth for both.
type Client struct{}

// New checks that the AWS CLI is available. It does not validate
// credentials; that's the aws CLI's job at call time.
func New() (*Client, error) {
	if _, err := exec.LookPath("aws"); err != nil {
		return nil, fmt.Errorf("aws CLI not found in PATH: %w", err)
	}
	return &Client{}, nil
}

// runAWS runs `aws <args...> --output json` and unmarshals stdout into out
// (if non-nil).
func runAWS(ctx context.Context, out any, args ...string) error {
	cmd := exec.CommandContext(ctx, "aws", append(args, "--output", "json")...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aws %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(stdout.Bytes(), out)
}
