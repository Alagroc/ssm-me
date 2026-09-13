package kubectl

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

type Node struct {
	Name   string
	Status string
	Labels map[string]string
}

func GetNodes() ([]Node, error) {
	out, err := exec.Command("kubectl", "get", "nodes", "--show-labels", "--no-headers").Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl get nodes: %w", err)
	}
	return ParseNodes(string(out)), nil
}

func GetCurrentContext() (string, error) {
	out, err := exec.Command("kubectl", "config", "current-context").Output()
	if err != nil {
		return "unknown", nil
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

func FilterNodes(nodes []Node, filters map[string]string) []Node {
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

func matchesAll(n Node, filters map[string]string) bool {
	for k, v := range filters {
		if got, ok := n.Labels[k]; !ok || got != v {
			return false
		}
	}
	return true
}

func ParseFilters(s string) map[string]string {
	f := make(map[string]string)
	for _, pair := range strings.Split(s, ",") {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) == 2 && kv[0] != "" {
			f[kv[0]] = kv[1]
		}
	}
	return f
}

func TopNode(name string) (string, error) {
	out, err := exec.Command("kubectl", "top", "node", name).Output()
	if err != nil {
		return "", fmt.Errorf("kubectl top node %s: %w", name, err)
	}
	return string(out), nil
}
