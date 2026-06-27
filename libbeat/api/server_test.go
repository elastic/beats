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
	"encoding/base64"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestConfiguration(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	if runtime.GOOS != "windows" {
		t.Skip("Check for User and Security Descriptor")
		return
	}
	t.Run("when user is set", func(t *testing.T) {
		cfg := config.MustNewConfigFrom(map[string]any{
			"host": "unix:///tmp/ok",
			"user": "admin",
		})

		_, err := New(logger, cfg)
		require.Error(t, err)
	})

	t.Run("when security descriptor is set", func(t *testing.T) {
		cfg := config.MustNewConfigFrom(map[string]any{
			"host":                "unix:///tmp/ok",
			"security_descriptor": "D:P(A;;GA;;;1234)",
		})

		_, err := New(logger, cfg)
		require.Error(t, err)
	})
}

func TestSocket(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

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

		cfg := config.MustNewConfigFrom(map[string]any{
			"host": "unix://" + sockFile,
		})

		s, err := New(logger, cfg)
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

		r, err := c.Get("http://unix/echo-hello") //nolint:noctx //Safe to not use ctx in test
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

		cfg := config.MustNewConfigFrom(map[string]any{
			"host": "unix://" + sockFile,
		})

		s, err := New(logger, cfg)
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

		r, err := c.Get("http://unix/echo-hello") //nolint:noctx //Safe to not use ctx in test
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

	cfg := config.MustNewConfigFrom(map[string]any{
		"host": url,
	})
	logger := logptest.NewTestingLogger(t, "")
	s, err := New(logger, cfg)
	require.NoError(t, err)
	attachEchoHelloHandler(t, s)
	go s.Start()
	defer func() {
		require.NoError(t, s.Stop())
	}()

	r, err := http.Get("http://" + s.l.Addr().String() + "/echo-hello") //nolint:noctx //Safe to not use ctx in test
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
	cfg := config.MustNewConfigFrom(map[string]any{
		"host": "http://localhost:0",
	})

	logger := logptest.NewTestingLogger(t, "")
	s, err := New(logger, cfg)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "http://"+s.l.Addr().String()+"/test", nil)

	// Test the first handler is attached.
	err = s.AttachHandler("/test", newTestHandler("test!"))
	require.NoError(t, err)
	resp := httptest.NewRecorder()
	s.mux.ServeHTTP(resp, req)
	assert.Equal(t, "test!", resp.Body.String())

	// Test the handler redirects properly.
	req = httptest.NewRequest(http.MethodGet, "http://"+s.l.Addr().String()+"/test/", nil)
	resp = httptest.NewRecorder()
	s.mux.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusMovedPermanently, resp.Result().StatusCode)
}

func TestOrdering(t *testing.T) {
	monitorSocket := genSocketPath()
	var monitorHost string
	if runtime.GOOS == "windows" {
		monitorHost = "npipe:///" + filepath.Base(monitorSocket)
	} else {
		monitorHost = "unix://" + monitorSocket
	}
	cfg := config.MustNewConfigFrom(map[string]any{
		"host": monitorHost,
	})

	t.Run("NewStartStop", func(t *testing.T) {
		defer goleak.VerifyNone(t)
		logger := logptest.NewTestingLogger(t, "")
		s, err := New(logger, cfg)
		require.NoError(t, err)
		s.Start()
		err = s.Stop()
		require.NoError(t, err)
		s.wg.Wait()
	})
	t.Run("NewStopStart", func(t *testing.T) {
		defer goleak.VerifyNone(t)
		logger := logptest.NewTestingLogger(t, "")
		s, err := New(logger, cfg)
		require.NoError(t, err)
		err = s.Stop()
		require.NoError(t, err)
		s.Start()
		s.wg.Wait()
	})
	t.Run("NewStop", func(t *testing.T) {
		defer goleak.VerifyNone(t)
		logger := logptest.NewTestingLogger(t, "")
		s, err := New(logger, cfg)
		require.NoError(t, err)
		err = s.Stop()
		require.NoError(t, err)
		s.wg.Wait()
	})
	t.Run("NewStopStop", func(t *testing.T) {
		defer goleak.VerifyNone(t)
		logger := logptest.NewTestingLogger(t, "")
		s, err := New(logger, cfg)
		require.NoError(t, err)
		err = s.Stop()
		require.NoError(t, err)
		err = s.Stop()
		require.NoError(t, err)
		s.wg.Wait()
	})
	t.Run("NewStartStartStop", func(t *testing.T) {
		defer goleak.VerifyNone(t)
		logger := logptest.NewTestingLogger(t, "")
		s, err := New(logger, cfg)
		require.NoError(t, err)
		s.Start()
		s.Start()
		err = s.Stop()
		require.NoError(t, err)
		s.wg.Wait()
	})
}

func newTestHandler(response string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, response)
	})
}

func genSocketPath() string {
	randData := make([]byte, 16)
	for i := range len(randData) {
		randData[i] = uint8(rand.UintN(255)) //nolint:gosec // 0-255 fits in a uint8
	}
	socketName := base64.URLEncoding.EncodeToString(randData) + ".sock"
	// don't use t.TempDir() because it can be too long
	socketDir := os.TempDir()
	return filepath.Join(socketDir, socketName)
}

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		host        string
		port        int
		wantNetwork string
		wantAddr    string
		errContains string
	}{
		{
			name:        "bare host uses tcp with port",
			host:        "localhost",
			port:        5066,
			wantNetwork: "tcp",
			wantAddr:    "localhost:5066",
		},
		{
			name:        "http scheme uses tcp with provided address",
			host:        "http://127.0.0.1:9200",
			port:        5066,
			wantNetwork: "tcp",
			wantAddr:    "127.0.0.1:9200",
		},
		{
			name:        "unix scheme uses socket path",
			host:        "unix:///tmp/beats.sock",
			port:        5066,
			wantNetwork: "unix",
			wantAddr:    "/tmp/beats.sock",
		},
		{
			name:        "unknown scheme returns error",
			host:        "ftp://localhost",
			port:        5066,
			errContains: "unknown scheme ftp",
		},
		{
			name:        "invalid URL returns parse error",
			host:        "http://[::1",
			port:        5066,
			errContains: "missing ']'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			network, addr, err := parse(tc.host, tc.port)
			if tc.errContains != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.errContains)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantNetwork, network)
			assert.Equal(t, tc.wantAddr, addr)
		})
	}
}
