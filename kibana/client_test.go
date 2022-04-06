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
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

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
