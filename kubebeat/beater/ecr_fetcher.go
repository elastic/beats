package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
)

const ECRType = "aws-ecr"

type ECRFetcher struct {
	ecrProvider *ECRProvider
}

func NewECRFetcher(cfg aws.Config) (Fetcher, error) {
	ecr := NewEcrProvider(cfg)

	return &ECRFetcher{
		ecrProvider: ecr,
	}, nil
}

func (f ECRFetcher) Fetch() ([]FetcherResult, error) {
	results := make([]FetcherResult, 0)
	ctx := context.Background()
	// TODO - The provider should get a list of the repositories it needs to check, and not check the entire ECR account`
	repositories, err := f.ecrProvider.DescribeAllECRRepositories(ctx)
	results = append(results, FetcherResult{ECRType, repositories})

	return results, err
}

func (f ECRFetcher) Stop() {
}
