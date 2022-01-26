package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
)

const IAMType = "aws-iam"

type IAMFetcher struct {
	iamProvider *IAMProvider
	roleName    string
}

func NewIAMFetcher(cfg aws.Config, roleName string) (Fetcher, error) {
	iam := NewIAMProvider(cfg)

	return &IAMFetcher{
		iamProvider: iam,
		roleName:    roleName,
	}, nil
}

func (f IAMFetcher) Fetch() ([]FetcherResult, error) {
	results := make([]FetcherResult, 0)
	ctx := context.Background()
	result, err := f.iamProvider.GetIAMRolePermissions(ctx, f.roleName)
	results = append(results, FetcherResult{IAMType, result})

	return results, err
}

func (f IAMFetcher) Stop() {
}
