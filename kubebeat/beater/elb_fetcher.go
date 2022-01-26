package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
)

const ELBType = "aws-elb"

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

func (f ELBFetcher) Fetch() ([]FetcherResult, error) {
	results := make([]FetcherResult, 0)
	ctx := context.Background()
	result, err := f.elbProvider.DescribeLoadBalancer(ctx, f.lbNames)
	results = append(results, FetcherResult{ELBType, result})

	return results, err
}

func (f ELBFetcher) Stop() {
}
