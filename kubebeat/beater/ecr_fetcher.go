package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type ECRDataFetcher struct {
}

/// This method will return a maximum of 100 repository
/// If we will ever wish to change it, DescribeRepositories returns results in paginated manner
/// When repoNames is nil, it will describe all the existing repositories
func (e ECRDataFetcher) DescribeRepositories(cfg aws.Config, ctx context.Context, repoNames []string) ([]ecr.Repository, error) {
	svc := ecr.New(cfg)
	input := &ecr.DescribeRepositoriesInput{
		RepositoryNames: repoNames,
	}

	req := svc.DescribeRepositoriesRequest(input)
	response, err := req.Send(ctx)
	if err != nil {
		logp.Err("Failed to fetch repository:%s from ecr, error - %+v", repoNames, err)
		return nil, err
	}

	return response.Repositories, err
}
