package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
)

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

func (f EKSFetcher) Fetch() ([]interface{}, error) {
	results := make([]interface{}, 0)
	ctx := context.Background()
	result, err := f.eksProvider.DescribeCluster(ctx, f.clusterName)
	results = append(results, result)

	return results, err
}

func (f EKSFetcher) Stop() {
}
