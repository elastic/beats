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

package integration_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elastic/beats/v9/libbeat/tests/integration"
)

func TestDisablingProxy(t *testing.T) {
	teapotMsg := http.StatusText(http.StatusTeapot)
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot)
			_, _ = w.Write([]byte(teapotMsg))
		}))

	proxy, proxyURL := integration.NewDisablingProxy(t, server.URL)
	checkStatusCodeAndBody(t, proxyURL, teapotMsg, http.StatusTeapot)

	proxy.Disable()
	checkStatusCodeAndBody(
		t,
		proxyURL,
		"Proxy is disabled\n",
		http.StatusServiceUnavailable)

	proxy.Enable()
	checkStatusCodeAndBody(t, proxyURL, teapotMsg, http.StatusTeapot)
}

func checkStatusCodeAndBody(t *testing.T, srvURL, body string, statusCode int) {
	t.Helper()

	resp, err := http.Get(srvURL) //nolint:gosec,noctx // It's a test
	if err != nil {
		t.Fatalf("could not call proxy: %s", err)
	}
	defer resp.Body.Close()
	if got, want := resp.StatusCode, statusCode; got != want {
		t.Fatalf("unexpected status code, got %d, want %d", got, want)
	}

	gotBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("cannot read response body: %s", err)
	}

	gotBody := string(gotBodyBytes)
	if gotBody != body {
		t.Fatalf("expecting body %q, got %q", body, gotBody)
	}
}
