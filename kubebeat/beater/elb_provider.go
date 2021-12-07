package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type ElbProvider struct {
}

func (provider ElbProvider) DescribeLoadBalancer(cfg aws.Config, ctx context.Context, balancersNames []string) (*elasticloadbalancing.DescribeLoadBalancersOutput, error) {
	svc := elasticloadbalancing.NewFromConfig(cfg)
	input := &elasticloadbalancing.DescribeLoadBalancersInput{
		LoadBalancerNames: balancersNames,
	}

	// TODO - There is next marker for large responses
	response, err := svc.DescribeLoadBalancers(ctx, input)
	if err != nil {
		logp.Err("Failed to describe cluster %s from ecr, error - %+v", balancersNames, err)
		return nil, err
	}

	return response, err
}
