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

package json

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestJSON(t *testing.T) {
	runner := compose.TestRunner{Service: "http"}

	runner.Run(t, compose.Suite{
		"FetchObject": testFetchObject,
		"FetchArray":  testFetchArray,
		"Data":        testData,
	})
}

func testFetchObject(t *testing.T, r compose.R) {
	f := mbtest.NewEventsFetcher(t, getConfig("object", r.Host()))
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)
}

func testFetchArray(t *testing.T, r compose.R) {
	f := mbtest.NewEventsFetcher(t, getConfig("array", r.Host()))
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)
}

func testData(t *testing.T, r compose.R) {
	f := mbtest.NewEventsFetcher(t, getConfig("object", r.Host()))
	err := mbtest.WriteEvents(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig(jsonType string, host string) map[string]interface{} {
	var path string
	var responseIsArray bool
	switch jsonType {
	case "object":
		path = "/jsonobj"
		responseIsArray = false
	case "array":
		path = "/jsonarr"
		responseIsArray = true
	}

	return map[string]interface{}{
		"module":        "http",
		"metricsets":    []string{"json"},
		"hosts":         []string{host},
		"path":          path,
		"namespace":     "testnamespace",
		"json.is_array": responseIsArray,
	}
}
