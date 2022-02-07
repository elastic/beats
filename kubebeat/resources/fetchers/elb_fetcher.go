package fetchers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/elastic/beats/v7/cloudbeat/resources"
)

const ELBType = "aws-elb"

type ELBFetcher struct {
	cfg         ELBFetcherConfig
	elbProvider *ELBProvider
}

type ELBFetcherConfig struct {
	resources.BaseFetcherConfig
	LoadBalancerNames []string `config:"loadBalancers"`
}

func NewELBFetcher(awsCfg aws.Config, cfg ELBFetcherConfig) (resources.Fetcher, error) {
	elb := NewELBProvider(awsCfg)

	return &ELBFetcher{
		elbProvider: elb,
		cfg:         cfg,
	}, nil
}

func (f ELBFetcher) Fetch(ctx context.Context) ([]resources.FetcherResult, error) {
	results := make([]resources.FetcherResult, 0)

	result, err := f.elbProvider.DescribeLoadBalancer(ctx, f.cfg.LoadBalancerNames)
	results = append(results, resources.FetcherResult{
		Type:     ELBType,
		Resource: result,
	})

	return results, err
}

func (f ELBFetcher) Stop() {
}
