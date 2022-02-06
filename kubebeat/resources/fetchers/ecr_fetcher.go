package fetchers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/elastic/beats/v7/kubebeat/resources"
)

const ECRType = "aws-ecr"

type ECRFetcher struct {
	cfg         ECRFetcherConfig
	ecrProvider *ECRProvider
}

type ECRFetcherConfig struct {
	resources.BaseFetcherConfig
}

func NewECRFetcher(awsCfg aws.Config, cfg ECRFetcherConfig) (resources.Fetcher, error) {
	ecr := NewEcrProvider(awsCfg)

	return &ECRFetcher{
		cfg:         cfg,
		ecrProvider: ecr,
	}, nil
}

func (f ECRFetcher) Fetch(ctx context.Context) ([]resources.FetcherResult, error) {
	results := make([]resources.FetcherResult, 0)

	// TODO - The provider should get a list of the repositories it needs to check, and not check the entire ECR account`
	repositories, err := f.ecrProvider.DescribeAllECRRepositories(ctx)
	results = append(results, resources.FetcherResult{
		Type:     ECRType,
		Resource: repositories,
	})

	return results, err
}

func (f ECRFetcher) Stop() {
}
