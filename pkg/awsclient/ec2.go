package awsclient

import (
	"context"
	"encoding/json"
	"fmt"
)

type ec2Filter struct {
	Name   string   `json:"Name"`
	Values []string `json:"Values"`
}

type describeInstancesOutput struct {
	Reservations []struct {
		Instances []struct {
			InstanceId     string `json:"InstanceId"`
			PrivateDnsName string `json:"PrivateDnsName"`
		} `json:"Instances"`
	} `json:"Reservations"`
}

// ResolveInstanceIDs is the EC2 fallback used when a node has no
// kubectl.InstanceIDLabel: it returns a map of private DNS name -> EC2
// instance ID via `aws ec2 describe-instances`.
func (c *Client) ResolveInstanceIDs(ctx context.Context, nodeNames []string) (map[string]string, error) {
	if len(nodeNames) == 0 {
		return map[string]string{}, nil
	}
	filters, err := json.Marshal([]ec2Filter{
		{Name: "private-dns-name", Values: nodeNames},
		{Name: "instance-state-name", Values: []string{"running"}},
	})
	if err != nil {
		return nil, err
	}

	var out describeInstancesOutput
	if err := runAWS(ctx, &out, "ec2", "describe-instances", "--filters", string(filters)); err != nil {
		return nil, fmt.Errorf("describe instances: %w", err)
	}

	result := make(map[string]string, len(nodeNames))
	for _, r := range out.Reservations {
		for _, inst := range r.Instances {
			if inst.PrivateDnsName != "" && inst.InstanceId != "" {
				result[inst.PrivateDnsName] = inst.InstanceId
			}
		}
	}
	return result, nil
}
