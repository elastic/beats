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

// +build !integration

package index

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"

<<<<<<< HEAD
=======
	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/mb"
>>>>>>> 4accfa821 (Introduce httpcommon package in libbeat (add support for Proxy) (#25219))
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)

var info = elasticsearch.Info{
	ClusterID:   "1234",
	ClusterName: "helloworld",
}

func TestMapper(t *testing.T) {
<<<<<<< HEAD
	elasticsearch.TestMapperWithInfo(t, "../index/_meta/test/stats.*.json", eventsMapping)
=======
	t.Skip("Skipping to fix in a follow up")

	mux := createEsMuxer("7.6.0", "platinum", false)

	server := httptest.NewServer(mux)
	defer server.Close()

	httpClient, err := helper.NewHTTPFromConfig(helper.Config{
		ConnectTimeout: 30 * time.Second,
		Transport: httpcommon.HTTPTransportSettings{
			Timeout: 30 * time.Second,
		},
	}, mb.HostData{
		URI:          server.URL,
		SanitizedURI: server.URL,
		Host:         server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	elasticsearch.TestMapperWithHttpHelper(t, "../index/_meta/test/stats.*.json", httpClient, eventsMapping)
>>>>>>> 4accfa821 (Introduce httpcommon package in libbeat (add support for Proxy) (#25219))
}

func TestEmpty(t *testing.T) {
	input, err := ioutil.ReadFile("./_meta/test/empty.512.json")
	require.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	eventsMapping(reporter, info, input)
	require.Equal(t, 0, len(reporter.GetEvents()))
}
