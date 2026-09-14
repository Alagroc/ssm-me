package kubectl

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/Alagroc/ssm-me/pkg/debuglog"
)

// InstanceIDLabel is the node label carrying the EC2/managed-instance ID
// directly, avoiding an EC2 API lookup when present.
const InstanceIDLabel = "topology.gemini.com/instance-id"

type Node struct {
	Name   string
	Status string
	Labels map[string]string
}

// runKubectl runs kubectl and logs the full command line, stdout, and
// stderr to the debug log, since errors surfaced in the TUI status bar get
// truncated.
func runKubectl(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	debuglog.Printf("kubectl %s\nstdout: %s\nstderr: %s\nerr: %v",
		strings.Join(args, " "), stdout.String(), strings.TrimSpace(stderr.String()), err)

	if err != nil {
		return nil, fmt.Errorf("kubectl %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func GetNodes(ctx context.Context) ([]Node, error) {
	out, err := runKubectl(ctx, "get", "nodes", "--show-labels", "--no-headers")
	if err != nil {
		return nil, err
	}
	return ParseNodes(string(out)), nil
}

func GetCurrentContext(ctx context.Context) (string, error) {
	out, err := runKubectl(ctx, "config", "current-context")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func ParseNodes(output string) []Node {
	var nodes []Node
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		nodes = append(nodes, Node{
			Name:   fields[0],
			Status: fields[1],
			Labels: parseLabels(fields[len(fields)-1]),
		})
	}
	return nodes
}

func parseLabels(s string) map[string]string {
	labels := make(map[string]string)
	if s == "<none>" {
		return labels
	}
	for _, pair := range strings.Split(s, ",") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			labels[kv[0]] = kv[1]
		}
	}
	return labels
}

// Filter matches nodes against a label. A Filter with a non-empty Value
// requires an exact key=value match; a Filter with an empty Value matches
// any node with a label key containing Key as a substring, so filtering by
// "topology.gemini.com" matches every "topology.gemini.com/*" label.
type Filter struct {
	Key   string
	Value string
}

func FilterNodes(nodes []Node, filters []Filter) []Node {
	if len(filters) == 0 {
		return nodes
	}
	var out []Node
	for _, n := range nodes {
		if matchesAll(n, filters) {
			out = append(out, n)
		}
	}
	return out
}

func matchesAll(n Node, filters []Filter) bool {
	for _, f := range filters {
		if len(MatchingLabels(n, f)) == 0 {
			return false
		}
	}
	return true
}

// MatchingLabels returns the label(s) on n that satisfy f.
func MatchingLabels(n Node, f Filter) map[string]string {
	matched := make(map[string]string)
	if f.Value != "" {
		if got, ok := n.Labels[f.Key]; ok && got == f.Value {
			matched[f.Key] = got
		}
		return matched
	}
	for k, v := range n.Labels {
		if strings.Contains(k, f.Key) {
			matched[k] = v
		}
	}
	return matched
}

// MatchedLabels returns the union, across all filters, of labels on n that
// caused it to match — useful for showing why a node is in the result set.
func MatchedLabels(n Node, filters []Filter) []string {
	matched := make(map[string]string)
	for _, f := range filters {
		for k, v := range MatchingLabels(n, f) {
			matched[k] = v
		}
	}
	keys := make([]string, 0, len(matched))
	for k := range matched {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, len(keys))
	for i, k := range keys {
		pairs[i] = k + "=" + matched[k]
	}
	return pairs
}

func ParseFilters(s string) []Filter {
	var filters []Filter
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		if kv[0] == "" {
			continue
		}
		f := Filter{Key: kv[0]}
		if len(kv) == 2 {
			f.Value = kv[1]
		}
		filters = append(filters, f)
	}
	return filters
}

func TopNode(name string) (string, error) {
	out, err := runKubectl(context.Background(), "top", "node", name)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
