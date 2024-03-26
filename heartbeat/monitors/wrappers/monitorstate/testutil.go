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

package monitorstate

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegtest"

	"github.com/elastic/beats/v7/heartbeat/esutil"
	"github.com/elastic/go-elasticsearch/v8"
)

// Helpers for tests here and elsewhere

func IntegESLoader(t *testing.T, esc *eslegclient.Connection, indexPattern string, location *config.LocationWithID) StateLoader {
	return MakeESLoader(esc, indexPattern, location)
}

func IntegES(t *testing.T) (esc *eslegclient.Connection) {
	conn, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
		URL:      eslegtest.GetURL(),
		Username: "admin",
		Password: "testing",
	})
	if err != nil {
		t.Fatal(err)
		panic(err) // panic in case TestLogger did not stop test
	}

	conn.Encoder = eslegclient.NewJSONEncoder(nil, false)

	err = conn.Connect()
	if err != nil {
		t.Fatal(err)
		panic(err) // panic in case TestLogger did not stop test
	}

	return conn
}

func IntegApiClient(t *testing.T) (esc *elasticsearch.Client) {
	esc, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{eslegtest.GetURL()},
		Username:  "admin",
		Password:  "testing",
	})
	require.NoError(t, err)
	respBody, err := esc.Cluster.Health()
	healthRaw, err := esutil.CheckRetResp(respBody, err)
	require.NoError(t, err)

	healthResp := struct {
		Status string `json:"status"`
	}{}
	err = json.Unmarshal(healthRaw, &healthResp)
	require.NoError(t, err)
	require.Contains(t, []string{"green", "yellow"}, healthResp.Status)

	return esc
}
