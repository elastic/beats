package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"time"
)

func TestElbProvider(t *testing.T) {

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	elbProvider := ELBProvider{}

	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	clusterName := []string{"adda9cdc89b13452e92d48be46858d37"}
	results, err := elbProvider.DescribeLoadBalancer(cfg, ctx, clusterName)

	if err != nil {
		assert.Fail(t, "Couldn't retrieve data from ecr", err)
	}

	assert.NotEmpty(t, results)
}
