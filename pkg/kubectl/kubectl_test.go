package kubectl

import (
	"testing"
)

const sampleOutput = `ip-10-0-142-45.us-east-1.compute.internal   Ready    <none>   12m   v1.30.2   beta.kubernetes.io/arch=amd64,beta.kubernetes.io/instance-type=c6i.xlarge,beta.kubernetes.io/os=linux,karpenter.k8s.aws/instance-cpu=4,karpenter.k8s.aws/instance-family=c6i,karpenter.k8s.aws/instance-memory=8192,karpenter.k8s.aws/instance-size=xlarge,karpenter.sh/capacity-type=on-demand,karpenter.sh/nodepool=default,kubernetes.io/arch=amd64,kubernetes.io/hostname=ip-10-0-142-45.us-east-1.compute.internal,kubernetes.io/os=linux,topology.kubernetes.io/region=us-east-1,topology.kubernetes.io/zone=us-east-1a
ip-10-0-200-10.us-east-1.compute.internal   Ready    <none>   10m   v1.30.2   beta.kubernetes.io/arch=amd64,beta.kubernetes.io/instance-type=m5.large,beta.kubernetes.io/os=linux,karpenter.sh/capacity-type=spot,karpenter.sh/nodepool=default,kubernetes.io/arch=amd64,kubernetes.io/hostname=ip-10-0-200-10.us-east-1.compute.internal,kubernetes.io/os=linux,topology.kubernetes.io/region=us-east-1,topology.kubernetes.io/zone=us-east-1b`

func TestParseNodes(t *testing.T) {
	nodes := ParseNodes(sampleOutput)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	n := nodes[0]
	if n.Name != "ip-10-0-142-45.us-east-1.compute.internal" {
		t.Errorf("name: got %q", n.Name)
	}
	if n.Status != "Ready" {
		t.Errorf("status: got %q", n.Status)
	}
	if n.Labels["beta.kubernetes.io/instance-type"] != "c6i.xlarge" {
		t.Errorf("instance-type label: %v", n.Labels)
	}
	if n.Labels["karpenter.sh/capacity-type"] != "on-demand" {
		t.Errorf("capacity-type label: %v", n.Labels)
	}
}

func TestParseNodesEmpty(t *testing.T) {
	nodes := ParseNodes("")
	if len(nodes) != 0 {
		t.Errorf("expected empty, got %d", len(nodes))
	}
}

func TestFilterByCapacityType(t *testing.T) {
	nodes := ParseNodes(sampleOutput)

	onDemand := FilterNodes(nodes, map[string]string{"karpenter.sh/capacity-type": "on-demand"})
	if len(onDemand) != 1 {
		t.Errorf("expected 1 on-demand node, got %d", len(onDemand))
	}
	if onDemand[0].Name != "ip-10-0-142-45.us-east-1.compute.internal" {
		t.Errorf("wrong node: %s", onDemand[0].Name)
	}

	spot := FilterNodes(nodes, map[string]string{"karpenter.sh/capacity-type": "spot"})
	if len(spot) != 1 {
		t.Errorf("expected 1 spot node, got %d", len(spot))
	}
}

func TestFilterByMultipleLabels(t *testing.T) {
	nodes := ParseNodes(sampleOutput)

	filtered := FilterNodes(nodes, map[string]string{
		"karpenter.sh/nodepool":      "default",
		"karpenter.sh/capacity-type": "on-demand",
	})
	if len(filtered) != 1 {
		t.Errorf("expected 1, got %d", len(filtered))
	}
}

func TestFilterNoMatch(t *testing.T) {
	nodes := ParseNodes(sampleOutput)
	filtered := FilterNodes(nodes, map[string]string{"env": "production"})
	if len(filtered) != 0 {
		t.Errorf("expected 0, got %d", len(filtered))
	}
}

func TestFilterEmpty(t *testing.T) {
	nodes := ParseNodes(sampleOutput)
	filtered := FilterNodes(nodes, map[string]string{})
	if len(filtered) != 2 {
		t.Errorf("expected all nodes with empty filter, got %d", len(filtered))
	}
}

func TestParseFilters(t *testing.T) {
	f := ParseFilters("karpenter.sh/capacity-type=on-demand,topology.kubernetes.io/zone=us-east-1a")
	if f["karpenter.sh/capacity-type"] != "on-demand" {
		t.Errorf("capacity-type: %v", f)
	}
	if f["topology.kubernetes.io/zone"] != "us-east-1a" {
		t.Errorf("zone: %v", f)
	}
}

func TestParseFiltersEmpty(t *testing.T) {
	f := ParseFilters("")
	if len(f) != 0 {
		t.Errorf("expected empty, got %v", f)
	}
}
