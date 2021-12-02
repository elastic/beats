package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"time"
)

func TestIamFetcherFetchRolePolicies(t *testing.T) {

	role := "chime-poc-NodeInstanceRole-ZI3XYU5TCY9X"
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	feather := IamFetcher{}

	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()


	results, err := feather.GetIamRolePermissions(cfg, ctx, role)

	if err != nil {
		assert.Fail(t, "Couldn't retrieve data from ecr", err)
	}

	assert.NotEmpty(t, results)
}
