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

package add_cloud_metadata

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
)

const azInstanceIdentityDocument = `{
	"location": "eastus2",
	"name": "test-az-vm",
	"offer": "UbuntuServer",
	"osType": "Linux",
	"platformFaultDomain": "0",
	"platformUpdateDomain": "0",
	"publisher": "Canonical",
	"sku": "14.04.4-LTS",
	"version": "14.04.201605091",
	"vmId": "04ab04c3-63de-4709-a9f9-9ab8c0411d5e",
	"vmSize": "Standard_D3_v2",
	"subscriptionId": "5tfb04c3-63de-4709-a9f9-9ab8c0411d5e"
}`

func initAzureTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/metadata/instance/compute?api-version=2017-04-02" && r.Header.Get("Metadata") == "true" {
			w.Write([]byte(azInstanceIdentityDocument))
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
}

func TestRetrieveAzureMetadata(t *testing.T) {
	logp.TestingSetup()

	server := initAzureTestServer()
	defer server.Close()

	config, err := common.NewConfigFrom(map[string]interface{}{
		"host": server.Listener.Addr().String(),
	})
	if err != nil {
		t.Fatal(err)
	}

	p, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	actual, err := p.Run(&beat.Event{Fields: common.MapStr{}})
	if err != nil {
		t.Fatal(err)
	}

	expected := common.MapStr{
		"cloud": common.MapStr{
			"provider": "azure",
			"instance": common.MapStr{
				"id":   "04ab04c3-63de-4709-a9f9-9ab8c0411d5e",
				"name": "test-az-vm",
			},
			"machine": common.MapStr{
				"type": "Standard_D3_v2",
			},
			"account": common.MapStr{
				"id": "5tfb04c3-63de-4709-a9f9-9ab8c0411d5e",
			},
			"service": common.MapStr{
				"name": "Virtual Machines",
			},
			"region": "eastus2",
		},
	}
	assert.Equal(t, expected, actual.Fields)
}
