// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package licenser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common/cli"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/libbeat/outputs/outil"
)

const (
	elasticsearchHost = "localhost"
	elasticsearchPort = "9200"
)

func getTestClient() *elasticsearch.Client {
	host := "http://" + cli.GetEnvOr("ES_HOST", elasticsearchHost) + ":" + cli.GetEnvOr("ES_POST", elasticsearchPort)
	client, err := elasticsearch.NewClient(elasticsearch.ClientSettings{
		URL:              host,
		Index:            outil.MakeSelector(),
		Username:         "myelastic", // NOTE: I will refactor this in a followup PR
		Password:         "changeme",
		Timeout:          60 * time.Second,
		CompressionLevel: 3,
	}, nil)

	if err != nil {
		panic(err)
	}
	return client
}

// Sanity check for schema change on the HTTP response from a live Elasticsearch instance.
func TestElasticsearch(t *testing.T) {
	f := NewElasticFetcher(getTestClient())
	license, err := f.Fetch()
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, Trial, license.Get())
	assert.Equal(t, Trial, license.Type)
	assert.Equal(t, Active, license.Status)

	assert.NotEmpty(t, license.UUID)
}
