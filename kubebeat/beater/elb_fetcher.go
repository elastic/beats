package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
)

type ELBFetcher struct {
	elbProvider *ELBProvider
	lbNames     []string
}

func NewELBFetcher(cfg aws.Config, loadBalancersNames []string) (Fetcher, error) {
	elb := NewELBProvider(cfg)

	return &ELBFetcher{
		elbProvider: elb,
		lbNames:     loadBalancersNames,
	}, nil
}

func (f ELBFetcher) Fetch() ([]interface{}, error) {
	results := make([]interface{}, 0)
	ctx := context.Background()
	result, err := f.elbProvider.DescribeLoadBalancer(ctx, f.lbNames)
	results = append(results, result)

	return results, err
}

func (f ELBFetcher) Stop() {
}
