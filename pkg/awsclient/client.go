package awsclient

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type Client struct {
	SSM *ssm.Client
	EC2 *ec2.Client
}

func New(ctx context.Context) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return &Client{
		SSM: ssm.NewFromConfig(cfg),
		EC2: ec2.NewFromConfig(cfg),
	}, nil
}
