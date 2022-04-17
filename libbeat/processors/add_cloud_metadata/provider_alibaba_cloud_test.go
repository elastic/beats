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

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
)

func initECSTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/latest/meta-data/instance-id" {
			w.Write([]byte("i-wz9g2hqiikg0aliyun2b"))
			return
		}
		if r.RequestURI == "/latest/meta-data/region-id" {
			w.Write([]byte("cn-shenzhen"))
			return
		}
		if r.RequestURI == "/latest/meta-data/zone-id" {
			w.Write([]byte("cn-shenzhen-a"))
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
}

func TestRetrieveAlibabaCloudMetadata(t *testing.T) {
	logp.TestingSetup()

	server := initECSTestServer()
	defer server.Close()

	config, err := common.NewConfigFrom(map[string]interface{}{
		"providers": []string{"alibaba"},
		"host":      server.Listener.Addr().String(),
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
			"provider": "ecs",
			"instance": common.MapStr{
				"id": "i-wz9g2hqiikg0aliyun2b",
			},
			"region":            "cn-shenzhen",
			"availability_zone": "cn-shenzhen-a",
			"service": common.MapStr{
				"name": "ECS",
			},
		},
	}
	assert.Equal(t, expected, actual.Fields)
}
