package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type ECRDataFetcher struct {
}

func (e ECRDataFetcher) DescribeAllRepositories(cfg aws.Config, ctx context.Context, repoNames []string) ([]types.Repository, error) {
	svc := ecr.NewFromConfig(cfg)
	input := &ecr.DescribeRepositoriesInput{
		RepositoryNames: repoNames,
	}

	response, err := svc.DescribeRepositories(ctx, input)
	if err != nil {
		logp.Err("Failed to fetch %s from ecr, error - %+v", repoNames, err)
		return nil, err
	}

	return response.Repositories, err
}
