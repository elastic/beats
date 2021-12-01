package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"log"
	"time"
)

type AwsKubeFetcher struct {
	cfg        aws.Config
	ecrFetcher ECRDataFetcher
}

func NewAwsKubeFetcherFetcher() Fetcher {

	// Need to take it from the config
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	ecr := ECRDataFetcher{}
	return &AwsKubeFetcher{
		cfg : cfg,
		ecrFetcher : ecr,
	}
}

func (f AwsKubeFetcher) Fetch() ([]interface{}, error) {

	//Get Images for ECR
	results := make([]interface{}, 0)

	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	repo := []string{"test-repo"}
	repositories, err := f.ecrFetcher.DescribeAllRepositories(f.cfg, ctx, repo)

	results = append(results, repositories)

	return results, err
}

func (f AwsKubeFetcher) Stop() {

}


