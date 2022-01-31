package fetchers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/elastic/beats/v7/kubebeat/resources"
)

const EKSType = "aws-eks"

type EKSFetcher struct {
	eksProvider *EKSProvider
	clusterName string
}

func NewEKSFetcher(cfg aws.Config, clusterName string) (resources.Fetcher, error) {
	eks := NewEksProvider(cfg)

	return &EKSFetcher{
		eksProvider: eks,
		clusterName: clusterName,
	}, nil
}

func (f EKSFetcher) Fetch() ([]resources.FetcherResult, error) {
	results := make([]resources.FetcherResult, 0)
	ctx := context.Background()
	result, err := f.eksProvider.DescribeCluster(ctx, f.clusterName)
	results = append(results, resources.FetcherResult{
		Type:     EKSType,
		Resource: result,
	})

	return results, err
}

func (f EKSFetcher) Stop() {
}
