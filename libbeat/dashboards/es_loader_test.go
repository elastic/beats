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

package dashboards

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch/estest"
)

func TestImporter(t *testing.T) {
	logp.TestingSetup()

	dashboardsConfig := Config{
		KibanaIndex: ".kibana-test",
		File:        "testdata/testbeat-dashboards.zip",
		Beat:        "testbeat",
	}

	client := estest.GetTestingElasticsearch(t)
	major := client.GetVersion().Major

	if major == 6 || major == 7 {
		t.Skip("Skipping tests for Elasticsearch 6.x releases")
	}

	loader := ElasticsearchLoader{
		client: client,
		config: &dashboardsConfig,
	}

	err := loader.CreateKibanaIndex()

	assert.NoError(t, err)

	version, _ := common.NewVersion("5.0.0")

	imp, err := NewImporter(*version, &dashboardsConfig, loader)
	assert.NoError(t, err)

	err = imp.Import()
	assert.NoError(t, err)

	status, _, _ := client.Request("GET", "/.kibana-test/dashboard/1e4389f0-e871-11e6-911d-3f8ed6f72700", "", nil, nil)
	assert.Equal(t, 200, status)
}

func TestImporterEmptyBeat(t *testing.T) {
	logp.TestingSetup()

	dashboardsConfig := Config{
		KibanaIndex: ".kibana-test-nobeat",
		File:        "testdata/testbeat-dashboards.zip",
		Beat:        "",
	}

	client := estest.GetTestingElasticsearch(t)
	major := client.GetVersion().Major
	if major == 6 || major == 7 {
		t.Skip("Skipping tests for Elasticsearch 6.x releases")
	}

	loader := ElasticsearchLoader{
		client: client,
		config: &dashboardsConfig,
	}

	version, _ := common.NewVersion("5.0.0")

	imp, err := NewImporter(*version, &dashboardsConfig, loader)
	assert.NoError(t, err)

	err = imp.Import()
	assert.NoError(t, err)

	status, _, _ := client.Request("GET", "/.kibana-test-nobeat/dashboard/1e4389f0-e871-11e6-911d-3f8ed6f72700", "", nil, nil)
	assert.Equal(t, 200, status)
}
