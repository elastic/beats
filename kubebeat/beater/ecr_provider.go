package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type ECRProvider struct {
	client *ecr.Client
}

func NewEcrProvider(cfg aws.Config) *ECRProvider {
	svc := ecr.New(cfg)
	return &ECRProvider{
		client: svc,
	}
}

// DescribeAllECRRepositories / This method will return a maximum of 100 repository
/// If we will ever wish to change it, DescribeRepositories returns results in paginated manner
func (provider ECRProvider) DescribeAllECRRepositories(ctx context.Context) ([]ecr.Repository, error) {
	/// When repoNames is nil, it will describe all the existing repositories
	return provider.DescribeRepositories(ctx, nil)
}

// DescribeRepositories / This method will return a maximum of 100 repository
/// If we will ever wish to change it, DescribeRepositories returns results in paginated manner
/// When repoNames is nil, it will describe all the existing repositories
func (provider ECRProvider) DescribeRepositories(ctx context.Context, repoNames []string) ([]ecr.Repository, error) {
	input := &ecr.DescribeRepositoriesInput{
		RepositoryNames: repoNames,
	}
	req := provider.client.DescribeRepositoriesRequest(input)
	response, err := req.Send(ctx)
	if err != nil {
		logp.Err("Failed to fetch repository:%s from ecr, error - %+v", repoNames, err)
		return nil, err
	}

	return response.Repositories, err
}
