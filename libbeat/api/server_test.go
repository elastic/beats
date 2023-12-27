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

package api

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/config"
)

func TestConfiguration(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Check for User and Security Descriptor")
		return
	}
	t.Run("when user is set", func(t *testing.T) {
		cfg := config.MustNewConfigFrom(map[string]interface{}{
			"host": "unix:///tmp/ok",
			"user": "admin",
		})

		_, err := New(nil, cfg)
		require.Error(t, err)
	})

	t.Run("when security descriptor is set", func(t *testing.T) {
		cfg := config.MustNewConfigFrom(map[string]interface{}{
			"host":                "unix:///tmp/ok",
			"security_descriptor": "D:P(A;;GA;;;1234)",
		})

		_, err := New(nil, cfg)
		require.Error(t, err)
	})
}

func TestSocket(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix Sockets don't work under windows")
		return
	}

	client := func(sockFile string) http.Client {
		return http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", sockFile)
				},
			},
		}
	}

	t.Run("socket doesn't exist before", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "testsocket")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		sockFile := tmpDir + "/test.sock"
		t.Log(sockFile)

		cfg := config.MustNewConfigFrom(map[string]interface{}{
			"host": "unix://" + sockFile,
		})

		s, err := New(nil, cfg)
		require.NoError(t, err)
		attachEchoHelloHandler(t, s)
		go s.Start()
		defer func() {
			require.NoError(t, s.Stop())
			// Make we cleanup behind
			_, err := os.Stat(sockFile)
			require.Error(t, err)
			require.False(t, os.IsExist(err))
		}()

		c := client(sockFile)

		r, err := c.Get("http://unix/echo-hello")
		require.NoError(t, err)
		defer r.Body.Close()

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		assert.Equal(t, "ehlo!", string(body))
		fi, err := os.Stat(sockFile)
		require.NoError(t, err)
		assert.Equal(t, socketFileMode, fi.Mode().Perm())
	})

	t.Run("starting beat and recover a dangling socket file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "testsocket")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		sockFile := tmpDir + "/test.sock"

		// Create the socket before the server.
		f, err := os.Create(sockFile)
		require.NoError(t, err)
		f.Close()

		cfg := config.MustNewConfigFrom(map[string]interface{}{
			"host": "unix://" + sockFile,
		})

		s, err := New(nil, cfg)
		require.NoError(t, err)
		attachEchoHelloHandler(t, s)
		go s.Start()
		defer func() {
			require.NoError(t, s.Stop())
			// Make we cleanup behind
			_, err := os.Stat(sockFile)
			require.Error(t, err)
			require.False(t, os.IsExist(err))
		}()

		c := client(sockFile)

		r, err := c.Get("http://unix/echo-hello")
		require.NoError(t, err)
		defer r.Body.Close()

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		assert.Equal(t, "ehlo!", string(body))

		fi, err := os.Stat(sockFile)
		require.NoError(t, err)
		assert.Equal(t, socketFileMode, fi.Mode().Perm(), "incorrect mode for file %s", sockFile)
	})
}

func TestHTTP(t *testing.T) {
	// select a random free port.
	url := "http://localhost:0"

	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"host": url,
	})

	s, err := New(nil, cfg)
	require.NoError(t, err)
	attachEchoHelloHandler(t, s)
	go s.Start()
	defer func() {
		require.NoError(t, s.Stop())
	}()

	r, err := http.Get("http://" + s.l.Addr().String() + "/echo-hello")
	require.NoError(t, err)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)

	assert.Equal(t, "ehlo!", string(body))
}

func attachEchoHelloHandler(t *testing.T, s *Server) {
	t.Helper()

	if err := s.AttachHandler("/echo-hello", newTestHandler("ehlo!")); err != nil {
		t.Fatal(err)
	}
}

func TestAttachHandler(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"host": "http://localhost:0",
	})

	s, err := New(nil, cfg)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "http://"+s.l.Addr().String()+"/test", nil)

	// Test the first handler is attached.
	err = s.AttachHandler("/test", newTestHandler("test!"))
	require.NoError(t, err)
	resp := httptest.NewRecorder()
	s.mux.ServeHTTP(resp, req)
	assert.Equal(t, "test!", resp.Body.String())

	// Handlers are matched in order so the first one will take precedence.
	err = s.AttachHandler("/test", newTestHandler("NOT test!"))
	require.NoError(t, err)
	resp = httptest.NewRecorder()
	s.mux.ServeHTTP(resp, req)
	assert.Equal(t, "test!", resp.Body.String())
}

func newTestHandler(response string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, response)
	})
}
