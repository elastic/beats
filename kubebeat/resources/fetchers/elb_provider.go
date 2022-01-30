package fetchers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type ELBProvider struct {
	client *elasticloadbalancing.Client
}

func NewELBProvider(cfg aws.Config) *ELBProvider {
	svc := elasticloadbalancing.New(cfg)
	return &ELBProvider{
		client: svc,
	}
}

/// DescribeLoadBalancer method will return up to 400 results
/// If we will ever want to increase this number, DescribeLoadBalancers support paginated requests
func (provider ELBProvider) DescribeLoadBalancer(ctx context.Context, balancersNames []string) ([]elasticloadbalancing.LoadBalancerDescription, error) {
	input := &elasticloadbalancing.DescribeLoadBalancersInput{
		LoadBalancerNames: balancersNames,
	}

	req := provider.client.DescribeLoadBalancersRequest(input)
	response, err := req.Send(ctx)
	if err != nil {
		logp.Err("Failed to describe load balancers %s from elb, error - %+v", balancersNames, err)
		return nil, err
	}

	return response.LoadBalancerDescriptions, err
}
