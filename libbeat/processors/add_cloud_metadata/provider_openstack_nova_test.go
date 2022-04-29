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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func openstackNovaMetadataHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == osMetadataInstanceIDURI {
			w.Write([]byte("i-0000ffac"))
			return
		}
		if r.RequestURI == osMetadataInstanceTypeURI {
			w.Write([]byte("m1.xlarge"))
			return
		}
		if r.RequestURI == osMetadataHostnameURI {
			w.Write([]byte("testvm01.stack.cloud"))
			return
		}
		if r.RequestURI == osMetadataZoneURI {
			w.Write([]byte("az-test-2"))
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	})
}

func TestRetrieveOpenstackNovaMetadata(t *testing.T) {
	logp.TestingSetup()

	server := httptest.NewServer(openstackNovaMetadataHandler())
	defer server.Close()

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"host": server.Listener.Addr().String(),
	})

	if err != nil {
		t.Fatal(err)
	}

	assertOpenstackNova(t, config)
}

func TestRetrieveOpenstackNovaMetadataWithHTTPS(t *testing.T) {
	logp.TestingSetup()

	server := httptest.NewTLSServer(openstackNovaMetadataHandler())
	defer server.Close()

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"host":                  server.Listener.Addr().String(),
		"ssl.verification_mode": "none",
	})

	if err != nil {
		t.Fatal(err)
	}

	assertOpenstackNova(t, config)
}

func assertOpenstackNova(t *testing.T, config *conf.C) {
	p, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	actual, err := p.Run(&beat.Event{Fields: mapstr.M{}})
	if err != nil {
		t.Fatal(err)
	}

	expected := mapstr.M{
		"cloud": mapstr.M{
			"provider": "openstack",
			"instance": mapstr.M{"" +
				"id": "i-0000ffac",
				"name": "testvm01.stack.cloud",
			},
			"machine": mapstr.M{
				"type": "m1.xlarge",
			},
			"availability_zone": "az-test-2",
			"service": mapstr.M{
				"name": "Nova",
			},
		},
	}
	assert.Equal(t, expected, actual.Fields)
}
