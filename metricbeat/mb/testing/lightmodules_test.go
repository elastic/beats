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

//go:build !integration
// +build !integration

package testing

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/metricbeat/mb"

	// Processor in the light module
	_ "github.com/elastic/beats/v8/libbeat/processors/actions"

	// Input used in the light module
	_ "github.com/elastic/beats/v8/metricbeat/module/http/json"
)

func init() {
	// To be moved to some kind of helper
	os.Setenv("BEAT_STRICT_PERMS", "false")
	mb.Registry.SetSecondarySource(mb.NewLightModulesSource("./testdata/lightmodules"))
}

func TestFetchLightModuleWithProcessors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintln(w, `{"foo":"bar"}`)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"module":     "test",
		"metricsets": []string{"json"},
		"hosts":      []string{ts.URL},
		"namespace":  "test",
	}
	ms := NewFetcher(t, config)
	events, errs := ms.FetchEvents()
	assert.Empty(t, errs)
	assert.NotEmpty(t, events)

	expected := common.MapStr{
		"http": common.MapStr{
			"test": common.MapStr{
				"foo": "bar",
			},
		},
		"service": common.MapStr{
			"type": "test",
		},

		// From the processor in the light module
		"fields": common.MapStr{
			"test": "fromprocessor",
		},
	}
	event := StandardizeEvent(ms.(mb.MetricSet), events[0])
	assert.EqualValues(t, expected, event.Fields)
}

func TestDataLightModuleWithProcessors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintln(w, `{"foo":"bar"}`)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"module":     "test",
		"metricsets": []string{"json"},
		"hosts":      []string{ts.URL},
		"namespace":  "test",
	}
	ms := NewFetcher(t, config)
	events, errs := ms.FetchEvents()
	assert.Empty(t, errs)
	assert.NotEmpty(t, events)

	dir, err := ioutil.TempDir("", "_meta-*")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	dataPath := filepath.Join(dir, "data.json")

	ms.WriteEvents(t, dataPath)

	var event struct {
		Event struct {
			Dataset string `json:"dataset"`
		} `json:"event"`
		Http struct {
			Test struct {
				Foo string `json:"foo"`
			} `json:"test"`
		} `json:"http"`

		// From the processor in the light module
		Fields struct {
			Test string `json:"test"`
		}
	}

	d, err := ioutil.ReadFile(dataPath)
	require.NoError(t, err)

	err = json.Unmarshal(d, &event)
	require.NoError(t, err)

	assert.Equal(t, "http.test", event.Event.Dataset)
	assert.Equal(t, "bar", event.Http.Test.Foo)
	assert.Equal(t, "fromprocessor", event.Fields.Test)
}
