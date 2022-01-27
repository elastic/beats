package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
)

const EKSType = "aws-eks"

type EKSFetcher struct {
	eksProvider *EKSProvider
	clusterName string
}

func NewEKSFetcher(cfg aws.Config, clusterName string) (Fetcher, error) {
	eks := NewEksProvider(cfg)

	return &EKSFetcher{
		eksProvider: eks,
		clusterName: clusterName,
	}, nil
}

func (f EKSFetcher) Fetch() ([]FetcherResult, error) {
	results := make([]FetcherResult, 0)
	ctx := context.Background()
	result, err := f.eksProvider.DescribeCluster(ctx, f.clusterName)
	results = append(results, FetcherResult{EKSType, result})

	return results, err
}

func (f EKSFetcher) Stop() {
}
