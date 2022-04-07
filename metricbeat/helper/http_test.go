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
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/metricbeat/helper/dialer"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/mb/parse"
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
	cfg.Transport.Timeout = 1 * time.Millisecond
	hostData := mb.HostData{
		URI:          ts.URL,
		SanitizedURI: ts.URL,
	}

	h, err := NewHTTPFromConfig(cfg, hostData)
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

	h, err := NewHTTPFromConfig(cfg, hostData)
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
	h, err := NewHTTPFromConfig(cfg, hostData)
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
	h, err = NewHTTPFromConfig(cfg, hostData)
	require.NoError(t, err)

	response, err = h.FetchResponse()
	response.Body.Close()
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode, "response status code")
}

func TestSetHeader(t *testing.T) {
	cfg := defaultConfig()
	cfg.Headers = map[string]string{
		"Override": "default",
	}

	h, err := NewHTTPFromConfig(cfg, mb.HostData{})
	require.NoError(t, err)

	h.SetHeader("Override", "overridden")
	v := h.headers.Get("override")
	assert.Equal(t, "overridden", v)
}

func TestSetHeaderDefault(t *testing.T) {
	cfg := defaultConfig()
	cfg.Headers = map[string]string{
		"Override": "default",
	}

	h, err := NewHTTPFromConfig(cfg, mb.HostData{})
	require.NoError(t, err)

	h.SetHeaderDefault("Override", "overridden")
	v := h.headers.Get("override")
	assert.Equal(t, "default", v)
}

func TestOverUnixSocket(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skipf("unix domain socket aren't supported under Windows")
		return
	}

	cases := map[string]struct {
		hostDataBuilder func(sockFile string) (mb.HostData, error)
	}{
		"at root": {
			hostDataBuilder: func(sockFile string) (mb.HostData, error) {
				return mb.HostData{
					Transport:    dialer.NewUnixDialerBuilder(sockFile),
					URI:          "http://unix/",
					SanitizedURI: "http://unix",
				}, nil
			},
		},
		"at specific path": {
			hostDataBuilder: func(sockFile string) (mb.HostData, error) {
				uri := "http://unix/ok"
				return mb.HostData{
					Transport:    dialer.NewUnixDialerBuilder(sockFile),
					URI:          uri,
					SanitizedURI: uri,
				}, nil
			},
		},
		"with parser builder": {
			hostDataBuilder: func(sockFile string) (mb.HostData, error) {
				parser := parse.URLHostParserBuilder{}.Build()
				return parser(&dummyModule{}, "http+unix://"+sockFile)
			},
		},
	}

	serveOnUnixSocket := func(t *testing.T, path string) net.Listener {
		l, err := net.Listen("unix", path)
		require.NoError(t, err)

		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "ehlo!")
		})

		go http.Serve(l, mux)

		return l
	}

	for title, c := range cases {
		t.Run(title, func(t *testing.T) {
			tmpDir, err := ioutil.TempDir("", "testsocket")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			sockFile := tmpDir + "/test.sock"
			l := serveOnUnixSocket(t, sockFile)
			defer l.Close()

			cfg := defaultConfig()

			hostData, err := c.hostDataBuilder(sockFile)
			require.NoError(t, err)

			h, err := NewHTTPFromConfig(cfg, hostData)
			require.NoError(t, err)

			r, err := h.FetchResponse()
			require.NoError(t, err)
			defer r.Body.Close()
			content, err := ioutil.ReadAll(r.Body)
			require.NoError(t, err)
			assert.Equal(t, []byte("ehlo!"), content)
		})
	}
}

func TestUserAgentCheck(t *testing.T) {
	ua := ""
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := defaultConfig()
	hostData := mb.HostData{
		URI:          ts.URL,
		SanitizedURI: ts.URL,
	}

	h, err := NewHTTPFromConfig(cfg, hostData)
	require.NoError(t, err)

	res, err := h.FetchResponse()
	require.NoError(t, err)
	res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, ua, "Metricbeat")
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

type dummyModule struct{}

func (*dummyModule) Name() string {
	return "dummy"
}

func (*dummyModule) Config() mb.ModuleConfig {
	return mb.ModuleConfig{}
}

func (*dummyModule) UnpackConfig(interface{}) error {
	return nil
}
