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

//go:build integration

package monitorstate

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/heartbeat/esutil"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/processors/util"
)

func TestStatesESLoader(t *testing.T) {
	testStart := time.Now()
	etc := newESTestContext(t)

	// Create three monitors in ES, load their states, and make sure we track them correctly
	// We create a few to make sure the query isolates the monitors correctly
	// and alternate between testing monitors that start up or down
	for i := 0; i < 10; i++ {
		testStatus := StatusUp
		if i%2 == 1 {
			testStatus = StatusDown
		}

		monID := etc.createTestMonitorStateInES(t, testStatus)
		// Since we've continued this state it should register the initial state
		ms := etc.tracker.GetCurrentState(monID, RetryConfig{})
		require.True(t, ms.StartedAt.After(testStart.Add(-time.Nanosecond)), "timestamp for new state is off")
		requireMSStatusCount(t, ms, testStatus, 1)

		// Write the state a few times, enough to guarantee a stable state
		count := FlappingThreshold * 2
		var lastId string
		for i := 0; i < count; i++ {
			ms = etc.tracker.RecordStatus(monID, testStatus, true)
			if i == 0 {
				lastId = ms.ID
			}
			require.Equal(t, lastId, ms.ID, "state ID should not change within state")
		}
		// The initial state adds 1 to count
		requireMSStatusCount(t, ms, testStatus, count+1)

		// now change the state
		if testStatus == StatusUp {
			testStatus = StatusDown
		} else {
			testStatus = StatusUp
		}

		origMsId := ms.ID
		for i := 0; i < count; i++ {
			ms = etc.tracker.RecordStatus(monID, testStatus, true)
			require.NotEqual(t, origMsId, ms.ID)
			if i == 0 {
				lastId = ms.ID
				require.Equal(t, origMsId, ms.Ends.ID, "transition should point to the prior state")
			}
			require.Equal(t, lastId, ms.ID, "state ID should not change within state")
		}
		requireMSStatusCount(t, ms, testStatus, count)
	}
}

func TestMakeESLoaderError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{
			name:       "should return a retryable error",
			statusCode: http.StatusInternalServerError,
			expected:   true,
		},
		{
			name:       "should not return a retryable error",
			statusCode: http.StatusNotFound,
			expected:   false,
		},
		{
			name:       "should not return a retryable error when handling malformed data",
			statusCode: http.StatusOK,
			expected:   false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			etc := newESTestContext(t)
			etc.ec.HTTP = fakeHTTPClient{respStatus: test.statusCode}
			loader := MakeESLoader(etc.ec, "fakeIndexPattern", etc.location)

			_, err := loader(stdfields.StdMonitorFields{})

			var loaderError LoaderError
			require.ErrorAs(t, err, &loaderError)
			require.Equal(t, loaderError.Retry, test.expected)
		})
	}
}

type fakeHTTPClient struct {
	respStatus int
}

func (fc fakeHTTPClient) Do(req *http.Request) (resp *http.Response, err error) {
	return &http.Response{
		StatusCode: fc.respStatus,
		Body:       io.NopCloser(strings.NewReader("test response")),
	}, nil
}

func (fc fakeHTTPClient) CloseIdleConnections() {
	// noop
}

type esTestContext struct {
	namespace string
	ec        *eslegclient.Connection
	esc       *elasticsearch.Client
	loader    StateLoader
	tracker   *Tracker
	location  *config.LocationWithID
}

func newESTestContext(t *testing.T) *esTestContext {
	location := &config.LocationWithID{
		ID: "TestId",
		Geo: util.GeoConfig{
			Name: "TestGeoName",
		},
	}
	namespace, _ := uuid.NewV4()
	esc := IntegApiClient(t)
	ec := IntegES(t)
	etc := &esTestContext{
		namespace: namespace.String(),
		esc:       esc,
		ec:        ec,
		loader:    IntegESLoader(t, ec, fmt.Sprintf("synthetics-*-%s", namespace.String()), location),
		location:  location,
	}

	etc.tracker = NewTracker(etc.loader, true)

	return etc
}

func (etc *esTestContext) createTestMonitorStateInES(t *testing.T, s StateStatus) (sf stdfields.StdMonitorFields) {
	mUUID, _ := uuid.NewV4()
	sf = stdfields.StdMonitorFields{
		ID:   mUUID.String(),
		Type: "test_type",
	}

	initState := newMonitorState(sf, s, 0, true)
	// Test int64 is un/marshalled correctly
	initState.DurationMs = 3e9
	etc.setInitialState(t, sf, initState)
	return sf
}

func (etc *esTestContext) setInitialState(t *testing.T, sf stdfields.StdMonitorFields, ms *State) {
	idx := fmt.Sprintf("synthetics-%s-%s", sf.Type, etc.namespace)

	type Mon struct {
		Id   string `json:"id"`
		Type string `json:"type"`
	}

	reqBodyRdr, err := esutil.ToJsonRdr(struct {
		Ts      time.Time `json:"@timestamp"`
		Monitor Mon       `json:"monitor"`
		State   *State    `json:"state"`
	}{
		Ts:      time.Now(),
		Monitor: Mon{Id: sf.ID, Type: sf.Type},
		State:   ms,
	})
	require.NoError(t, err)

	_, err = esutil.CheckRetResp(etc.esc.Index(idx, reqBodyRdr, func(request *esapi.IndexRequest) {
		// Refresh the index since we tend to re-query immediately, otherwise this would miss
		request.Refresh = "true"

	}))
	require.NoError(t, err)
}
