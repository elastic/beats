package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"time"
)

func TestEksDataFetcherFetchECR(t *testing.T) {

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	eksFetcher := ECRDataFetcher{}

	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	repoNames := []string{"test-repo"}
	results, err := eksFetcher.DescribeAllRepositories(cfg, ctx, repoNames)

	if err != nil {
		assert.Fail(t, "Couldn't retrieve data from ecr", err)
	}

	assert.NotEmpty(t, results)
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
