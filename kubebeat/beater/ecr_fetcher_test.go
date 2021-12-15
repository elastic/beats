package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"time"
)

func TestEksDataFetcherFetchECR(t *testing.T) {

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		log.Fatal(err)
	}
	eksFetcher := ECRDataFetcher{}

	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	repoNames := []string{"test-repo", "amazon-k8s-cn"}

	results, err := eksFetcher.DescribeRepositories(cfg, ctx, repoNames)

	if err != nil {
		assert.Fail(t, "Couldn't retrieve data from ecr", err)
	}

	assert.NotEmpty(t, results)
}
