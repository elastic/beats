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

package transport

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestCloseConnectionError(t *testing.T) {
	// Set up a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"status": "ok"}`)
	}))
	defer server.Close()

	logger, observedLogs := logptest.NewTestingLoggerWithObserver(t, "test")
	// Set IdleConnTimeout to 2 seconds and a custom dialer
	transport := &http.Transport{
		IdleConnTimeout: 2 * time.Second,
		// uses our implementation of custom dialer
		DialContext: LoggingDialer(NetDialer(10*time.Second), logger).DialContext,
	}

	client := &http.Client{
		Transport: transport,
	}

	// First request to the test server
	resp, err := client.Get(server.URL) //nolint:noctx // It is a test
	require.NoError(t, err, "first request failed")
	_, _ = io.ReadAll(resp.Body)
	resp.Body.Close()

	// Wait for a duration longer than IdleConnTimeout
	waitTime := 6 * time.Second
	time.Sleep(waitTime)

	// Second request to the test server after idle timeout
	resp, err = client.Get(server.URL) //nolint:noctx // It is a test
	require.NoError(t, err, "second request failed")
	_, _ = io.ReadAll(resp.Body)
	resp.Body.Close()

	logs := observedLogs.FilterMessageSnippet("Error reading from connection:").TakeAll()
	assert.Equal(t, 0, len(logs), "did not ignore use of closed connection error")

}
