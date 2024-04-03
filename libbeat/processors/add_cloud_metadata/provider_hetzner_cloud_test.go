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
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func hetznerMetadataHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == hetznerMetadataInstanceIDURI {
			_, _ = w.Write([]byte("111111"))
			return
		}
		if r.RequestURI == hetznerMetadataHostnameURI {
			_, _ = w.Write([]byte("my-hetzner-instance"))
			return
		}
		if r.RequestURI == hetznerMetadataAvailabilityZoneURI {
			_, _ = w.Write([]byte("hel1-dc2"))
			return
		}
		if r.RequestURI == hetznerMetadataRegionURI {
			_, _ = w.Write([]byte("eu-central"))
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}
}

func TestRetrieveHetznerMetadata(t *testing.T) {
	logp.TestingSetup()

	server := httptest.NewServer(hetznerMetadataHandler())
	defer server.Close()

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"host": server.Listener.Addr().String(),
	})

	if err != nil {
		t.Fatal(err)
	}

	assertHetzner(t, config)
}

func assertHetzner(t *testing.T, config *conf.C) {
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
			"provider": "hetzner",
			"instance": mapstr.M{"" +
				"id": "111111",
				"name": "my-hetzner-instance",
			},
			"availability_zone": "hel1-dc2",
			"region":            "eu-central",
			"service": mapstr.M{
				"name": "Cloud",
			},
		},
	}
	assert.Equal(t, expected, actual.Fields)
}
