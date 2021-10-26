// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build integration
// +build integration

package licenser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common/cli"
	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
)

const (
	elasticsearchHost = "localhost"
	elasticsearchPort = "9200"
)

func getTestClient() *eslegclient.Connection {
	transport := httpcommon.DefaultHTTPTransportSettings()
	transport.Timeout = 60 * time.Second

	host := "http://" + cli.GetEnvOr("ES_HOST", elasticsearchHost) + ":" + cli.GetEnvOr("ES_POST", elasticsearchPort)
	client, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
		URL:              host,
		Username:         "admin",
		Password:         "testing",
		CompressionLevel: 3,
		Transport:        transport,
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

	assert.Equal(t, Basic, license.Type)
	assert.Equal(t, Active, license.Status)
	assert.NotEmpty(t, license.UUID)
}
