package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"time"
)

func TestEksDataProvider(t *testing.T) {

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		log.Fatal(err)
	}
	eksProvider := EKSProvider{}

	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	clusterName := "EKS-Elastic-agent-demo"
	results, err := eksProvider.DescribeCluster(cfg, ctx, clusterName)

	if err != nil {
		assert.Fail(t, "Couldn't retrieve data from ecr", err)
	}

	assert.NotEmpty(t, results)
}
