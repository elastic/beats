package fetchers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/elastic/beats/v7/kubebeat/resources"
)

const IAMType = "aws-iam"

type IAMFetcher struct {
	iamProvider *IAMProvider
	roleName    string
}

func NewIAMFetcher(cfg aws.Config, roleName string) (resources.Fetcher, error) {
	iam := NewIAMProvider(cfg)

	return &IAMFetcher{
		iamProvider: iam,
		roleName:    roleName,
	}, nil
}

func (f IAMFetcher) Fetch() ([]resources.FetcherResult, error) {
	results := make([]resources.FetcherResult, 0)
	ctx := context.Background()
	result, err := f.iamProvider.GetIAMRolePermissions(ctx, f.roleName)
	results = append(results, resources.FetcherResult{
		Type:     IAMType,
		Resource: result,
	})

	return results, err
}

func (f IAMFetcher) Stop() {
}
