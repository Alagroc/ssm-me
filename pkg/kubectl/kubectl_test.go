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

	onDemand := FilterNodes(nodes, []Filter{{Key: "karpenter.sh/capacity-type", Value: "on-demand"}})
	if len(onDemand) != 1 {
		t.Errorf("expected 1 on-demand node, got %d", len(onDemand))
	}
	if onDemand[0].Name != "ip-10-0-142-45.us-east-1.compute.internal" {
		t.Errorf("wrong node: %s", onDemand[0].Name)
	}

	spot := FilterNodes(nodes, []Filter{{Key: "karpenter.sh/capacity-type", Value: "spot"}})
	if len(spot) != 1 {
		t.Errorf("expected 1 spot node, got %d", len(spot))
	}
}

func TestFilterByMultipleLabels(t *testing.T) {
	nodes := ParseNodes(sampleOutput)

	filtered := FilterNodes(nodes, []Filter{
		{Key: "karpenter.sh/nodepool", Value: "default"},
		{Key: "karpenter.sh/capacity-type", Value: "on-demand"},
	})
	if len(filtered) != 1 {
		t.Errorf("expected 1, got %d", len(filtered))
	}
}

func TestFilterNoMatch(t *testing.T) {
	nodes := ParseNodes(sampleOutput)
	filtered := FilterNodes(nodes, []Filter{{Key: "env", Value: "production"}})
	if len(filtered) != 0 {
		t.Errorf("expected 0, got %d", len(filtered))
	}
}

func TestFilterEmpty(t *testing.T) {
	nodes := ParseNodes(sampleOutput)
	filtered := FilterNodes(nodes, nil)
	if len(filtered) != 2 {
		t.Errorf("expected all nodes with empty filter, got %d", len(filtered))
	}
}

func TestFilterByLabelKeySubstring(t *testing.T) {
	nodes := ParseNodes(sampleOutput)

	filtered := FilterNodes(nodes, []Filter{{Key: "topology.kubernetes.io"}})
	if len(filtered) != 2 {
		t.Errorf("expected 2 nodes matching key substring, got %d", len(filtered))
	}

	filtered = FilterNodes(nodes, []Filter{{Key: "karpenter.k8s.aws"}})
	if len(filtered) != 1 || filtered[0].Name != "ip-10-0-142-45.us-east-1.compute.internal" {
		t.Errorf("expected 1 matching node, got %v", filtered)
	}
}

func TestMatchedLabels(t *testing.T) {
	nodes := ParseNodes(sampleOutput)
	n := nodes[0]

	got := MatchedLabels(n, []Filter{{Key: "karpenter.sh/capacity-type", Value: "on-demand"}})
	if len(got) != 1 || got[0] != "karpenter.sh/capacity-type=on-demand" {
		t.Errorf("expected exact-match label, got %v", got)
	}

	got = MatchedLabels(n, []Filter{{Key: "topology.kubernetes.io"}})
	want := []string{"topology.kubernetes.io/region=us-east-1", "topology.kubernetes.io/zone=us-east-1a"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("expected %v, got %v", want, got)
			break
		}
	}
}

func TestParseFilters(t *testing.T) {
	f := ParseFilters("karpenter.sh/capacity-type=on-demand,topology.kubernetes.io/zone=us-east-1a")
	if len(f) != 2 || f[0] != (Filter{Key: "karpenter.sh/capacity-type", Value: "on-demand"}) {
		t.Errorf("capacity-type: %v", f)
	}
	if f[1] != (Filter{Key: "topology.kubernetes.io/zone", Value: "us-east-1a"}) {
		t.Errorf("zone: %v", f)
	}
}

func TestParseFiltersKeyOnly(t *testing.T) {
	f := ParseFilters("topology.gemini.com")
	if len(f) != 1 || f[0] != (Filter{Key: "topology.gemini.com"}) {
		t.Errorf("expected key-only filter, got %v", f)
	}
}

func TestParseFiltersEmpty(t *testing.T) {
	f := ParseFilters("")
	if len(f) != 0 {
		t.Errorf("expected empty, got %v", f)
	}
}
