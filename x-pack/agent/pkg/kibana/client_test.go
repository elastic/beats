// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/logger"
)

func noopWrapper(rt http.RoundTripper) (http.RoundTripper, error) {
	return rt, nil
}

func addCatchAll(mux *http.ServeMux, t *testing.T) *http.ServeMux {
	mux.HandleFunc("/", func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("HTTP catch all handled called")
	})
	return mux
}

// - Prefix.
func TestHTTPClient(t *testing.T) {
	ctx := context.Background()
	l, err := logger.New()
	require.NoError(t, err)

	t.Run("Guard against double slashes on path", withServer(
		func(t *testing.T) *http.ServeMux {
			msg := `{ message: "hello" }`
			mux := http.NewServeMux()
			mux.HandleFunc("/nested/echo-hello", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, msg)
			})
			return addCatchAll(mux, t)
		}, func(t *testing.T, host string) {
			// Add a slashes at the end of the URL, internally we should prevent having double slashes
			// when adding path to the request.
			url := "http://" + host + "/"

			c, err := NewConfigFromURL(url)
			client, err := NewWithConfig(l, c, noopWrapper)

			require.NoError(t, err)
			resp, err := client.Send(ctx, "GET", "/nested/echo-hello", nil, nil, nil)
			require.NoError(t, err)

			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, `{ message: "hello" }`, string(body))
		},
	))

	t.Run("Simple call", withServer(
		func(t *testing.T) *http.ServeMux {
			msg := `{ message: "hello" }`
			mux := http.NewServeMux()
			mux.HandleFunc("/echo-hello", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, msg)
			})
			return mux
		}, func(t *testing.T, host string) {
			cfg := config.MustNewConfigFrom(map[string]interface{}{
				"host": host,
			})

			client, err := NewWithRawConfig(nil, cfg, nil)
			require.NoError(t, err)
			resp, err := client.Send(ctx, "GET", "/echo-hello", nil, nil, nil)
			require.NoError(t, err)

			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, `{ message: "hello" }`, string(body))
		},
	))

	t.Run("Simple call with a prefix path", withServer(
		func(t *testing.T) *http.ServeMux {
			msg := `{ message: "hello" }`
			mux := http.NewServeMux()
			mux.HandleFunc("/mycustompath/echo-hello", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, msg)
			})
			return mux
		}, func(t *testing.T, host string) {
			cfg := config.MustNewConfigFrom(map[string]interface{}{
				"host": host,
				"path": "mycustompath",
			})

			client, err := NewWithRawConfig(nil, cfg, nil)
			require.NoError(t, err)
			resp, err := client.Send(ctx, "GET", "/echo-hello", nil, nil, nil)
			require.NoError(t, err)

			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, `{ message: "hello" }`, string(body))
		},
	))

	t.Run("Basic auth when credentials are valid", withServer(
		func(t *testing.T) *http.ServeMux {
			msg := `{ message: "hello" }`
			mux := http.NewServeMux()
			mux.HandleFunc("/echo-hello", basicAuthHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, msg)
			}, "hello", "world", "testing"))
			return mux
		}, func(t *testing.T, host string) {
			cfg := config.MustNewConfigFrom(map[string]interface{}{
				"username": "hello",
				"password": "world",
				"host":     host,
			})

			client, err := NewWithRawConfig(nil, cfg, nil)
			require.NoError(t, err)
			resp, err := client.Send(ctx, "GET", "/echo-hello", nil, nil, nil)
			require.NoError(t, err)

			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, `{ message: "hello" }`, string(body))
		},
	))

	t.Run("Basic auth when credentials are invalid", withServer(
		func(t *testing.T) *http.ServeMux {
			msg := `{ message: "hello" }`
			mux := http.NewServeMux()
			mux.HandleFunc("/echo-hello", basicAuthHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, msg)
			}, "hello", "world", "testing"))
			return mux
		}, func(t *testing.T, host string) {
			cfg := config.MustNewConfigFrom(map[string]interface{}{
				"username": "bye",
				"password": "world",
				"host":     host,
			})

			client, err := NewWithRawConfig(nil, cfg, nil)
			require.NoError(t, err)
			resp, err := client.Send(ctx, "GET", "/echo-hello", nil, nil, nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		},
	))

	t.Run("Custom user agent", withServer(
		func(t *testing.T) *http.ServeMux {
			msg := `{ message: "hello" }`
			mux := http.NewServeMux()
			mux.HandleFunc("/echo-hello", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, msg)
				require.Equal(t, r.Header.Get("User-Agent"), "custom-agent")
			})
			return mux
		}, func(t *testing.T, host string) {
			cfg := config.MustNewConfigFrom(map[string]interface{}{
				"host": host,
			})

			client, err := NewWithRawConfig(nil, cfg, func(wrapped http.RoundTripper) (http.RoundTripper, error) {
				return NewUserAgentRoundTripper(wrapped, "custom-agent"), nil
			})

			require.NoError(t, err)
			resp, err := client.Send(ctx, "GET", "/echo-hello", nil, nil, nil)
			require.NoError(t, err)

			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, `{ message: "hello" }`, string(body))
		},
	))

	t.Run("Enforce Kibana version", withServer(
		func(t *testing.T) *http.ServeMux {
			msg := `{ message: "hello" }`
			mux := http.NewServeMux()
			mux.HandleFunc("/echo-hello", enforceKibanaHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, msg)
			}, "8.0.0"))
			return mux
		}, func(t *testing.T, host string) {
			cfg := config.MustNewConfigFrom(map[string]interface{}{
				"host": host,
			})

			client, err := NewWithRawConfig(nil, cfg, func(wrapped http.RoundTripper) (http.RoundTripper, error) {
				return NewEnforceKibanaVersionRoundTripper(wrapped, "8.0.0"), nil
			})

			require.NoError(t, err)
			resp, err := client.Send(ctx, "GET", "/echo-hello", nil, nil, nil)
			require.NoError(t, err)

			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, `{ message: "hello" }`, string(body))
		},
	))

	t.Run("Allows to debug HTTP request between a client and a server", withServer(
		func(t *testing.T) *http.ServeMux {
			msg := `{ "message": "hello" }`
			mux := http.NewServeMux()
			mux.HandleFunc("/echo-hello", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, msg)
			})
			return mux
		}, func(t *testing.T, host string) {

			debugger := &debugStack{}

			cfg := config.MustNewConfigFrom(map[string]interface{}{
				"host": host,
			})

			client, err := NewWithRawConfig(nil, cfg, func(wrapped http.RoundTripper) (http.RoundTripper, error) {
				return NewDebugRoundTripper(wrapped, debugger), nil
			})

			require.NoError(t, err)
			resp, err := client.Send(ctx, "GET", "/echo-hello", nil, nil, bytes.NewBuffer([]byte("hello")))
			require.NoError(t, err)

			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, `{ "message": "hello" }`, string(body))

			for _, m := range debugger.messages {
				fmt.Println(m)
			}

			assert.Equal(t, 1, len(debugger.messages))
		},
	))
}

func withServer(m func(t *testing.T) *http.ServeMux, test func(t *testing.T, host string)) func(t *testing.T) {
	return func(t *testing.T) {
		listener, err := net.Listen("tcp", ":0")
		require.NoError(t, err)
		defer listener.Close()

		port := listener.Addr().(*net.TCPAddr).Port

		go http.Serve(listener, m(t))

		test(t, "localhost:"+strconv.Itoa(port))
	}
}

func basicAuthHandler(handler http.HandlerFunc, username, password, realm string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()

		if !ok || u != username || p != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		handler(w, r)
	}
}

func enforceKibanaHandler(handler http.HandlerFunc, version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("kbn-version") != version {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		handler(w, r)
	}
}

type debugStack struct {
	sync.Mutex
	messages []string
}

func (d *debugStack) Debug(args ...interface{}) {
	d.Lock()
	defer d.Unlock()

	// This should not happen in testing.
	m, ok := args[0].(string)
	if !ok {
		panic("could not convert message to string ")
	}

	d.messages = append(d.messages, m)
}
