// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package licenser

import (
	"net/http"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/esleg/eslegclient"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common/cli"
	"github.com/elastic/beats/v7/libbeat/common/cli"
	"github.com/elastic/beats/v7/libbeat/outputs/transport"
)

const (
	elasticsearchHost = "localhost"
	elasticsearchPort = "9200"
)

func getTestClient() *eslegclient.Connection {
	host := "http://" + cli.GetEnvOr("ES_HOST", elasticsearchHost) + ":" + cli.GetEnvOr("ES_POST", elasticsearchPort)
	client, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
		URL:              host,
		Username:         "myelastic", // NOTE: I will refactor this in a followup PR
		Password:         "changeme",
		CompressionLevel: 3,
		HTTP: &http.Client{
			Transport: &http.Transport{
				Dial: transport.NetDialer(60 * time.Second).Dial,
			},
			Timeout: 60 * time.Second,
		},
	})

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
