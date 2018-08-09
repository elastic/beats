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
		Username:         cli.GetEnvOr("ES_USER", ""),
		Password:         cli.GetEnvOr("ES_PASS", ""),
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

	assert.NotNil(t, license.Get())
	assert.NotNil(t, license.Type)
	assert.Equal(t, Active, license.Status)

	assert.NotEmpty(t, license.UUID)

	assert.NotNil(t, license.Features.Graph)
	assert.NotNil(t, license.Features.Logstash)
	assert.NotNil(t, license.Features.ML)
	assert.NotNil(t, license.Features.Monitoring)
	assert.NotNil(t, license.Features.Rollup)
	assert.NotNil(t, license.Features.Security)
	assert.NotNil(t, license.Features.Watcher)
}
