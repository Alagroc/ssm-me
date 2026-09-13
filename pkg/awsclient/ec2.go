package awsclient

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// ResolveInstanceIDs returns a map of private DNS name -> EC2 instance ID.
func (c *Client) ResolveInstanceIDs(ctx context.Context, nodeNames []string) (map[string]string, error) {
	if len(nodeNames) == 0 {
		return map[string]string{}, nil
	}
	out, err := c.EC2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("private-dns-name"), Values: nodeNames},
			{Name: aws.String("instance-state-name"), Values: []string{"running"}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("describe instances: %w", err)
	}
	result := make(map[string]string, len(nodeNames))
	for _, r := range out.Reservations {
		for _, inst := range r.Instances {
			if inst.PrivateDnsName != nil && inst.InstanceId != nil {
				result[*inst.PrivateDnsName] = *inst.InstanceId
			}
		}
	}
	return result, nil
}
