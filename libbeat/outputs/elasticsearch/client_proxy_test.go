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

// This file contains tests to confirm that elasticsearch.Client uses proxy
// settings following the intended precedence.

package elasticsearch

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/outputs/outil"
)

// TestClientPing is a placeholder test that does nothing on a standard run,
// but starts up a client and sends a ping when the environment variable
// TEST_START_CLIENT is set to 1 as in execClient).
func TestClientPing(t *testing.T) {
	// If this is the child process, start up the client, otherwise do nothing.
	if os.Getenv("TEST_START_CLIENT") == "1" {
		doClientPing(t)
		return
	}
}

// TestBaseline makes sure we can have a client process ping the server that
// we start, with no changes to the proxy settings. (This is really a
// meta-test for the helpers that create the servers / client.)
func TestBaseline(t *testing.T) {
	listeners := testListeners{}
	listeners.start(t)
	defer listeners.close()

	// Start a bare client with no proxy settings, pointed at the main server.
	err := execClient("TEST_SERVER_URL=" + listeners.server.URL)
	assert.NoError(t, err)
	// We expect one server request and 0 proxy requests
	assert.Equal(t, 1, listeners.serverRequestCount())
	assert.Equal(t, 0, listeners.proxyRequestCount())
}

// TestClientSettingsProxy confirms that we can control the proxy of a client
// by setting its ClientSettings.Proxy value on creation. (The child process
// uses the TEST_PROXY_URL environment variable to initialize the flag.)
func TestClientSettingsProxy(t *testing.T) {
	listeners := testListeners{}
	listeners.start(t)
	defer listeners.close()

	// Start a client with ClientSettings.Proxy set to the proxy listener.
	err := execClient(
		"TEST_SERVER_URL="+listeners.server.URL,
		"TEST_PROXY_URL="+listeners.proxy.URL)
	assert.NoError(t, err)
	// We expect one proxy request and 0 server requests
	assert.Equal(t, 0, listeners.serverRequestCount())
	assert.Equal(t, 1, listeners.proxyRequestCount())
}

// TestEnvironmentProxy confirms that we can control the proxy of a client by
// setting the HTTP_PROXY environment variable (see
// https://golang.org/pkg/net/http/#ProxyFromEnvironment).
func TestEnvironmentProxy(t *testing.T) {
	listeners := testListeners{}
	listeners.start(t)
	defer listeners.close()

	// Start a client with HTTP_PROXY set to the proxy listener.
	// The server is set to a nonexistent URL because ProxyFromEnvironment
	// always returns a nil proxy for local destination URLs. For this case, we
	// confirm the intended destination by setting it in the http headers,
	// triggered in doClientPing by the TEST_HEADER_URL environment variable.
	err := execClient(
		"TEST_SERVER_URL=http://fakeurl.fake.not-real",
		"TEST_HEADER_URL="+listeners.server.URL,
		"HTTP_PROXY="+listeners.proxy.URL)
	assert.NoError(t, err)
	// We expect one proxy request and 0 server requests
	assert.Equal(t, 0, listeners.serverRequestCount())
	assert.Equal(t, 1, listeners.proxyRequestCount())
}

// TestClientSettingsOverrideEnvironmentProxy confirms that when both
// ClientSettings.Proxy and HTTP_PROXY are set, ClientSettings takes precedence.
func TestClientSettingsOverrideEnvironmentProxy(t *testing.T) {
	listeners := testListeners{}
	listeners.start(t)
	defer listeners.close()

	// Start a client with ClientSettings.Proxy set to the proxy listener and
	// HTTP_PROXY set to the server listener. We expect that the former will
	// override the latter and thus we will only see a ping to the proxy.
	// As above, the fake URL is needed to ensure ProxyFromEnvironment gives a
	// non-nil result.
	err := execClient(
		"TEST_SERVER_URL=http://fakeurl.fake.not-real",
		"TEST_HEADER_URL="+listeners.server.URL,
		"TEST_PROXY_URL="+listeners.proxy.URL,
		"HTTP_PROXY="+listeners.server.URL)
	assert.NoError(t, err)
	// We expect one proxy request and 0 server requests
	assert.Equal(t, 0, listeners.serverRequestCount())
	assert.Equal(t, 1, listeners.proxyRequestCount())
}

// runClientTest executes the current test binary as a child process,
// running only the TestClientPing, and calling it with the environment variable
// TEST_START_CLIENT=1 (so the test can recognize that it is the child process),
// and any additional environment settings specified in env.
// This is helpful for testing proxy settings, since we need to have both a
// proxy / server-side listener and a client that communicates with the server
// using various proxy settings.
func execClient(env ...string) error {
	// The child process always runs only the TestClientPing test, which pings
	// the server at TEST_SERVER_URL and then terminates.
	cmd := exec.Command(os.Args[0], "-test.run=TestClientPing")
	cmd.Env = append(append(os.Environ(),
		"TEST_START_CLIENT=1"),
		env...)
	return cmd.Run()
}

func doClientPing(t *testing.T) {
	serverURL := os.Getenv("TEST_SERVER_URL")
	proxy := os.Getenv("TEST_PROXY_URL")
	headerURL := os.Getenv("TEST_HEADER_URL")
	clientSettings := ClientSettings{
		URL:     serverURL,
		Index:   outil.MakeSelector(outil.ConstSelectorExpr("test")),
		Headers: map[string]string{"X-Test-URL": serverURL},
	}
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		require.NoError(t, err)
		clientSettings.Proxy = proxyURL
	}
	if headerURL != "" {
		// Some tests point at an intentionally invalid server because golang's
		// helpers never return a proxy for a request to the local host (see
		// TestEnvironmentProxy), and in that case we still want to record the real
		// server URL in the headers for verification.
		clientSettings.Headers["X-Test-URL"] = headerURL
	}
	client, err := NewClient(clientSettings, nil)
	require.NoError(t, err)

	// This ping won't succeed; we aren't testing end-to-end communication
	// (which would require a lot more setup work), we just want to make sure
	// the client is pointed at the right server or proxy.
	client.Ping()
}

// testListeners is a wrapper that handles starting up http listeners
// representing an elastic server and an intermediate proxy, confirming that
// requests target the expected (server) URL, and counting how many requests
// arrive at each endpoint.
type testListeners struct {
	server *httptest.Server
	proxy  *httptest.Server

	_serverRequestCount atomic.Int // Requests directly to the server
	_proxyRequestCount  atomic.Int // Requests via the proxy
}

// start creates and starts the server and proxy listeners.
func (tl *testListeners) start(t *testing.T) {
	tl.server = httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, tl.server.URL, r.Header.Get("X-Test-URL"))
			fmt.Fprintln(w, "Hello, client")
			tl._serverRequestCount.Inc()
		}))
	tl.proxy = httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, tl.server.URL, r.Header.Get("X-Test-URL"))
			fmt.Fprintln(w, "Hello, client")
			tl._proxyRequestCount.Inc()
		}))
}

func (tl *testListeners) close() {
	tl.server.Close()
	tl.proxy.Close()
}

// Convenience functions to unwrap the atomic primitives
func (tl testListeners) serverRequestCount() int {
	return tl._serverRequestCount.Load()
}

func (tl testListeners) proxyRequestCount() int {
	return tl._proxyRequestCount.Load()
}
