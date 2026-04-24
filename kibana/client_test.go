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

package kibana

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/config"
)

const (
	binaryName = "Testbeat"
	v          = "9.9.9"
	commit     = "1234abcd"
	buildTime  = "20001212"
)

func TestErrorJson(t *testing.T) {
	// also common 200: {"objects":[{"id":"apm-*","type":"index-pattern","error":{"message":"[doc][index-pattern:test-*]: version conflict, document already exists (current version [1])"}}]}
	kibanaTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message": "Cannot export dashboard", "attributes":{"objects":[{"id":"test-*","type":"index-pattern","error":{"message":"action [indices:data/write/bulk[s]] is unauthorized for user [test]"}}]}}`))
	}))
	defer kibanaTS.Close()

	assertConnection(t, kibanaTS.URL, http.StatusUnauthorized)
}

func assertConnection(t *testing.T, URL string, expectedStatusCode int) {
	t.Helper()
	conn := Connection{
		URL:  URL,
		HTTP: http.DefaultClient,
	}
	code, _, err := conn.Request(http.MethodPost, "", url.Values{}, nil, nil)
	assert.Equal(t, expectedStatusCode, code)
	assert.Error(t, err)
}

func TestIsServerless(t *testing.T) {
	rawStatusCall := `{"name":"kb","uuid":"d2130570-f7d8-463b-bd67-8503150004d5","version":{"number":"8.15.0","build_hash":"13382875e99e8c97f4574d86eca07cac3be9edfc","build_number":75422,"build_snapshot":false,"build_flavor":"stateful","build_date":"2024-06-15T18:13:50.595Z"}}`

	kibanaTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(rawStatusCall))
	}))
	defer kibanaTS.Close()

	conn := Connection{
		URL:  kibanaTS.URL,
		HTTP: http.DefaultClient,
	}

	testClient := Client{
		Connection: conn,
	}

	got, err := testClient.KibanaIsServerless()
	require.NoError(t, err)
	require.False(t, got)
}

func TestErrorBadJson(t *testing.T) {
	kibanaTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
		_, _ = w.Write([]byte(`{`))
	}))
	defer kibanaTS.Close()

	assertConnection(t, kibanaTS.URL, http.StatusGone)
}

func TestErrorJsonWithHTTPOK(t *testing.T) {
	kibanaTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"successCount":0,"success":false,"warnings":[],"errors":[{"id":"abcf35b0-0a82-11e8-bffe-ff7d4f68cf94-ecs","type":"dashboard","title":"[Filebeat MongoDB] Overview ECS","meta":{"title":"[Filebeat MongoDB] Overview ECS","icon":"dashboardApp"},"error":{"type":"missing_references","references":[{"type":"search","id":"e49fe000-0a7e-11e8-bffe-ff7d4f68cf94-ecs"},{"type":"search","id":"bfc96a60-0a80-11e8-bffe-ff7d4f68cf94-ecs"}]}}]}`))
	}))
	defer kibanaTS.Close()

	assertConnection(t, kibanaTS.URL, http.StatusOK)
}

func TestSuccess(t *testing.T) {
	kibanaTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"objects":[{"id":"test-*","type":"index-pattern","updated_at":"2018-01-24T19:04:13.371Z","version":1}]}`))

		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "bar", r.Header.Get("foo"))
	}))
	defer kibanaTS.Close()

	conn := Connection{
		URL:  kibanaTS.URL,
		HTTP: http.DefaultClient,
	}
	code, _, err := conn.Request(http.MethodPost, "", url.Values{}, http.Header{"foo": []string{"bar"}}, nil)
	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
}

func TestServiceToken(t *testing.T) {
	serviceToken := "fakeservicetoken"

	kibanaTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))

		assert.Equal(t, "Bearer "+serviceToken, r.Header.Get("Authorization"))
	}))
	defer kibanaTS.Close()

	conn := Connection{
		URL:          kibanaTS.URL,
		HTTP:         http.DefaultClient,
		ServiceToken: serviceToken,
	}
	code, _, err := conn.Request(http.MethodPost, "", url.Values{}, http.Header{"foo": []string{"bar"}}, nil)
	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
}

func TestNewKibanaClientWithSpace(t *testing.T) {
	var (
		testSpace      = "test-space"
		spaceURLPrefix = "/s/" + testSpace
	)

	var requests []*http.Request
	kibanaTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r)
		if r.URL.Path == spaceURLPrefix+statusAPI {
			_, _ = w.Write([]byte(`{"version":{"number":"1.2.3-beta","build_snapshot":true}}`))
		}
	}))
	defer kibanaTS.Close()

	// Configure an arbitrary test space to ensure the space URL prefix is added.
	client, err := NewKibanaClient(config.MustNewConfigFrom(fmt.Sprintf(`
protocol: http
host: %s
space.id: %s
headers:
  key: value
  content-type: text/plain
  accept: text/plain
  kbn-xsrf: 0
`, kibanaTS.Listener.Addr().String(), testSpace)), binaryName, v, commit, buildTime)
	require.NoError(t, err)
	require.NotNil(t, client)

	_, _, err = client.Request(http.MethodPost, "/foo", url.Values{}, http.Header{"key": []string{"another_value"}}, nil)
	require.NoError(t, err)

	// NewKibanaClient issues a request to /api/status to fetch the version.
	require.Len(t, requests, 2)
	assert.Equal(t, spaceURLPrefix+statusAPI, requests[0].URL.Path)
	assert.Equal(t, []string{"value"}, requests[0].Header.Values("key"))
	assert.Equal(t, "1.2.3-beta-SNAPSHOT", client.Version.String())

	// Headers specified in cient.Request are added to those defined in config.
	//
	// Content-Type, Accept, and kbn-xsrf cannot be overridden.
	assert.Equal(t, spaceURLPrefix+"/foo", requests[1].URL.Path)
	assert.Equal(t, []string{"value", "another_value"}, requests[1].Header.Values("key"))
	assert.Equal(t, []string{"application/json"}, requests[1].Header.Values("Content-Type"))
	assert.Equal(t, []string{"application/json"}, requests[1].Header.Values("Accept"))
	assert.Equal(t, []string{"1"}, requests[1].Header.Values("kbn-xsrf"))

}

func TestRetryOnStatus(t *testing.T) {
	var requestCount atomic.Int32
	kibanaTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusBadGateway) // 502 — triggers retry
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer kibanaTS.Close()

	conn := Connection{
		URL:  kibanaTS.URL,
		HTTP: http.DefaultClient,
		Retry: RetryConfig{
			MaxRetries:    3,
			RetryOnStatus: []int{502, 503, 504},
		},
	}
	code, _, err := conn.Request(http.MethodGet, "", nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, int32(3), requestCount.Load())
}

func TestRetryExhausted(t *testing.T) {
	var requestCount atomic.Int32
	kibanaTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable) // 503 — always retried
		_, _ = w.Write([]byte(`{"message":"unavailable"}`))
	}))
	defer kibanaTS.Close()

	conn := Connection{
		URL:  kibanaTS.URL,
		HTTP: http.DefaultClient,
		Retry: RetryConfig{
			MaxRetries:    2,
			RetryOnStatus: []int{503},
		},
	}
	// After MaxRetries exhausted, the last response is returned rather than an error.
	code, _, _ := conn.Request(http.MethodGet, "", nil, nil, nil)
	assert.Equal(t, http.StatusServiceUnavailable, code)
	assert.Equal(t, int32(3), requestCount.Load()) // 1 initial + 2 retries
}

func TestRetryDisabled(t *testing.T) {
	var requestCount atomic.Int32
	kibanaTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"message":"bad gateway"}`))
	}))
	defer kibanaTS.Close()

	conn := Connection{
		URL:  kibanaTS.URL,
		HTTP: http.DefaultClient,
		Retry: RetryConfig{
			MaxRetries:    0,
			RetryOnStatus: []int{502},
		},
	}
	_, _, _ = conn.Request(http.MethodGet, "", nil, nil, nil)
	assert.Equal(t, int32(1), requestCount.Load())
}

func TestRetryCustomRetryOnError(t *testing.T) {
	retryOnErrorCalled := false
	// A server that immediately closes connections forces a transport error.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hijack and close to simulate a connection reset.
		hj, ok := w.(http.Hijacker)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer ts.Close()

	conn := Connection{
		URL:  ts.URL,
		HTTP: http.DefaultClient,
		Retry: RetryConfig{
			MaxRetries: 1,
			RetryOnError: func(req *http.Request, err error) bool {
				retryOnErrorCalled = true
				return false // don't retry
			},
		},
	}
	resp, err := conn.Send(http.MethodGet, "", nil, nil, nil)
	if resp != nil {
		resp.Body.Close()
	}
	require.Error(t, err)
	assert.True(t, retryOnErrorCalled)
}

func TestRetryBackoff(t *testing.T) {
	var requestCount atomic.Int32
	kibanaTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		if n < 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer kibanaTS.Close()

	backoffCalled := false
	conn := Connection{
		URL:  kibanaTS.URL,
		HTTP: http.DefaultClient,
		Retry: RetryConfig{
			MaxRetries:    2,
			RetryOnStatus: []int{502},
			RetryBackoff: func(attempt int) time.Duration {
				backoffCalled = true
				assert.Equal(t, 1, attempt)
				return 0 // no actual sleep so the test stays fast
			},
		},
	}
	code, _, err := conn.Request(http.MethodGet, "", nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	assert.True(t, backoffCalled)
}

func TestRetryWithBody(t *testing.T) {
	const payload = `{"hello":"world"}`
	var requestCount atomic.Int32
	kibanaTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, payload, string(body), "body must be identical on every attempt")
		if n < 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer kibanaTS.Close()

	conn := Connection{
		URL:  kibanaTS.URL,
		HTTP: http.DefaultClient,
		Retry: RetryConfig{
			MaxRetries:    2,
			RetryOnStatus: []int{502},
		},
	}
	code, _, err := conn.Request(http.MethodPost, "", nil, nil, strings.NewReader(payload))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, int32(2), requestCount.Load())
}

func TestNewKibanaClientWithMultipartData(t *testing.T) {
	var requests []*http.Request
	kibanaTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r)
		if r.URL.Path == statusAPI {
			_, _ = w.Write([]byte(`{"version":{"number":"1.2.3-beta","build_snapshot":true}}`))
		}
	}))
	defer kibanaTS.Close()

	// Don't configure a space to ensure the space URL prefix is not added.
	client, err := NewKibanaClient(config.MustNewConfigFrom(fmt.Sprintf(`
protocol: http
host: %s
headers:
  content-type: multipart/form-data; boundary=46bea21be603a2c2ea6f51571a5e1baf5ea3be8ebd7101199320607b36ff
  accept: text/plain
  kbn-xsrf: 0
`, kibanaTS.Listener.Addr().String())), binaryName, v, commit, buildTime)
	require.NoError(t, err)
	require.NotNil(t, client)

	_, _, err = client.Request(http.MethodPost, "/foo", url.Values{}, http.Header{"key": []string{"another_value"}}, nil)
	require.NoError(t, err)

	assert.Equal(t, []string{"multipart/form-data; boundary=46bea21be603a2c2ea6f51571a5e1baf5ea3be8ebd7101199320607b36ff"}, requests[1].Header.Values("Content-Type"))

}
