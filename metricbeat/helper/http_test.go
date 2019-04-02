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

package helper

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/metricbeat/mb"
)

func TestGetAuthHeaderFromToken(t *testing.T) {
	tests := []struct {
		Name, Content, Expected string
	}{
		{
			"Test a token is read",
			"testtoken",
			"Bearer testtoken",
		},
		{
			"Test a token is trimmed",
			"testtoken\n",
			"Bearer testtoken",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			content := []byte(test.Content)
			tmpfile, err := ioutil.TempFile("", "token")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write(content); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			header, err := getAuthHeaderFromToken(tmpfile.Name())
			assert.NoError(t, err)
			assert.Equal(t, test.Expected, header)
		})
	}
}

func TestGetAuthHeaderFromTokenNoFile(t *testing.T) {
	header, err := getAuthHeaderFromToken("nonexistingfile")
	assert.Equal(t, "", header)
	assert.Error(t, err)
}

func TestTimeout(t *testing.T) {
	c := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-c:
		case <-r.Context().Done():
		}
	}))
	defer ts.Close()

	cfg := defaultConfig()
	cfg.Timeout = 1 * time.Millisecond
	hostData := mb.HostData{
		URI:          ts.URL,
		SanitizedURI: ts.URL,
	}

	h, err := newHTTPFromConfig(cfg, "test", hostData)
	require.NoError(t, err)

	checkTimeout(t, h)
	close(c)
}

func TestConnectTimeout(t *testing.T) {
	// This IP shouldn't exist, 192.0.2.0/24 is reserved for testing
	uri := "http://192.0.2.42"
	cfg := defaultConfig()
	cfg.ConnectTimeout = 1 * time.Nanosecond
	hostData := mb.HostData{
		URI:          uri,
		SanitizedURI: uri,
	}

	h, err := newHTTPFromConfig(cfg, "test", hostData)
	require.NoError(t, err)

	checkTimeout(t, h)
}

func TestAuthentication(t *testing.T) {
	expectedUser := "elastic"
	expectedPassword := "super1234"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok || user != expectedUser || password != expectedPassword {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer ts.Close()

	cfg := defaultConfig()

	// Unauthorized
	hostData := mb.HostData{
		URI:          ts.URL,
		SanitizedURI: ts.URL,
	}
	h, err := newHTTPFromConfig(cfg, "test", hostData)
	require.NoError(t, err)

	response, err := h.FetchResponse()
	response.Body.Close()
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode, "response status code")

	// Authorized
	hostData = mb.HostData{
		URI:          ts.URL,
		SanitizedURI: ts.URL,
		User:         expectedUser,
		Password:     expectedPassword,
	}
	h, err = newHTTPFromConfig(cfg, "test", hostData)
	require.NoError(t, err)

	response, err = h.FetchResponse()
	response.Body.Close()
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode, "response status code")
}

func checkTimeout(t *testing.T, h *HTTP) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		response, err := h.FetchResponse()
		assert.Error(t, err)
		if response != nil {
			response.Body.Close()
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout should have happened time ago")
	}
}
