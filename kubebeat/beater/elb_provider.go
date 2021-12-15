package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type ELBProvider struct {
}

/// DescribeLoadBalancer method will return up to 400 results
/// If we will ever want to increase this number, DescribeLoadBalancers support paginated requests
func (provider ELBProvider) DescribeLoadBalancer(cfg aws.Config, ctx context.Context, balancersNames []string) ([]elasticloadbalancing.LoadBalancerDescription, error) {
	svc := elasticloadbalancing.New(cfg)
	input := &elasticloadbalancing.DescribeLoadBalancersInput{
		LoadBalancerNames: balancersNames,
	}

	req := svc.DescribeLoadBalancersRequest(input)
	response, err := req.Send(ctx)
	if err != nil {
		logp.Err("Failed to describe cluster %s from ecr, error - %+v", balancersNames, err)
		return nil, err
	}

	return response.LoadBalancerDescriptions, err
}
