package fetchers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/elastic/beats/v7/kubebeat/resources"
)

const ELBType = "aws-elb"

type ELBFetcher struct {
	elbProvider *ELBProvider
	lbNames     []string
}

func NewELBFetcher(cfg aws.Config, loadBalancersNames []string) (resources.Fetcher, error) {
	elb := NewELBProvider(cfg)

	return &ELBFetcher{
		elbProvider: elb,
		lbNames:     loadBalancersNames,
	}, nil
}

func (f ELBFetcher) Fetch(ctx context.Context) ([]resources.FetcherResult, error) {
	results := make([]resources.FetcherResult, 0)

	result, err := f.elbProvider.DescribeLoadBalancer(ctx, f.lbNames)
	results = append(results, resources.FetcherResult{
		Type:     ELBType,
		Resource: result,
	})

	return results, err
}

func (f ELBFetcher) Stop() {
}
