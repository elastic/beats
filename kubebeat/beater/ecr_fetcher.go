package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/elastic/beats/v7/libbeat/logp"
	"time"
)

type ECRDataFetcher struct {
}

func (e ECRDataFetcher) DescribeAllRepositories(cfg aws.Config, ctx context.Context, repoNames []string) ([]types.Repository, error) {

	// Create context to enable explicit cancellation of the http requests.
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

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

////TODO - Remove
//func (e ECRDataFetcher) DescribeAllRepositories(repoNames []string) ([]types.Repository, error) {
//	// TODO - load configuration from config
//	cfg, err := config.LoadDefaultConfig(context.TODO())
//
//	if err != nil {
//		log.Fatal(err)
//	}
//	// Create context to enable explicit cancellation of the http requests.
//	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
//	defer cancel()
//
//	svc := ecr.NewFromConfig(cfg)
//	input := &ecr.DescribeRepositoriesInput{
//		RepositoryNames: repoNames,
//	}
//
//	response, err := svc.DescribeRepositories(ctx, input)
//	if err != nil {
//		logp.Err("Failed to fetch %s from ecr, error - %+v", repoName, err)
//		return nil, err
//	}
//
//	return response.Repositories, err
//}
