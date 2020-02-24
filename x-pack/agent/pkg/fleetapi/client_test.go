// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleetapi

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/agent/kibana"
	"github.com/elastic/beats/x-pack/agent/pkg/config"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/release"
)

func TestHTTPClient(t *testing.T) {
	ctx := context.Background()

	t.Run("Ensure we validate the remote Kibana version is higher or equal", withServer(
		func(t *testing.T) *http.ServeMux {
			msg := `{ message: "hello" }`
			mux := http.NewServeMux()
			mux.HandleFunc("/echo-hello", authHandler(func(w http.ResponseWriter, r *http.Request) {
				v := r.Header.Get("kbn-version")
				assert.Equal(t, release.Version(), v)
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, msg)
			}, "abc123"))
			return mux
		}, func(t *testing.T, host string) {
			cfg := &kibana.Config{
				Host: host,
			}

			l, err := logger.New()
			client, err := NewAuthWithConfig(l, "abc123", cfg)
			require.NoError(t, err)
			resp, err := client.Send(ctx, "GET", "/echo-hello", nil, nil, nil)
			require.NoError(t, err)

			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, `{ message: "hello" }`, string(body))
		},
	))

	t.Run("API Key is valid", withServer(
		func(t *testing.T) *http.ServeMux {
			msg := `{ message: "hello" }`
			mux := http.NewServeMux()
			mux.HandleFunc("/echo-hello", authHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, msg)
			}, "abc123"))
			return mux
		}, func(t *testing.T, host string) {
			cfg := config.MustNewConfigFrom(map[string]interface{}{
				"host": host,
			})

			client, err := kibana.NewWithRawConfig(nil, cfg, func(wrapped http.RoundTripper) (http.RoundTripper, error) {
				return NewFleetAuthRoundTripper(wrapped, "abc123")
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

	t.Run("API Key is not valid", withServer(
		func(t *testing.T) *http.ServeMux {
			msg := `{ message: "hello" }`
			mux := http.NewServeMux()
			mux.HandleFunc("/echo-hello", authHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, msg)
			}, "secret"))
			return mux
		}, func(t *testing.T, host string) {
			cfg := config.MustNewConfigFrom(map[string]interface{}{
				"host": host,
			})

			client, err := kibana.NewWithRawConfig(nil, cfg, func(wrapped http.RoundTripper) (http.RoundTripper, error) {
				return NewFleetAuthRoundTripper(wrapped, "abc123")
			})

			require.NoError(t, err)
			_, err = client.Send(ctx, "GET", "/echo-hello", nil, nil, nil)
			require.Error(t, err)
		},
	))

	t.Run("Fleet user agent", withServer(
		func(t *testing.T) *http.ServeMux {
			msg := `{ message: "hello" }`
			mux := http.NewServeMux()
			mux.HandleFunc("/echo-hello", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, msg)
				require.Equal(t, r.Header.Get("User-Agent"), "Beat Agent v8.0.0")
			})
			return mux
		}, func(t *testing.T, host string) {
			cfg := config.MustNewConfigFrom(map[string]interface{}{
				"host": host,
			})

			client, err := kibana.NewWithRawConfig(nil, cfg, func(wrapped http.RoundTripper) (http.RoundTripper, error) {
				return NewFleetUserAgentRoundTripper(wrapped, "8.0.0"), nil
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

	t.Run("Fleet endpoint is not responding", func(t *testing.T) {
		cfg := config.MustNewConfigFrom(map[string]interface{}{
			"host": "127.0.0.0:7278",
		})

		timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		client, err := kibana.NewWithRawConfig(nil, cfg, func(wrapped http.RoundTripper) (http.RoundTripper, error) {
			return NewFleetAuthRoundTripper(wrapped, "abc123")
		})

		_, err = client.Send(timeoutCtx, "GET", "/echo-hello", nil, nil, nil)
		require.Error(t, err)
	})
}

// NOTE(ph): Usually I would be against testing private methods as much as possible but in this
// case since we might deal with different format or error I make sense to test this method in
// isolation.
func TestExtract(t *testing.T) {
	// The error before is returned when an exception or an internal occur in Kibana, they
	// are not only generated by the Fleet app.
	t.Run("standard high level kibana errors", func(t *testing.T) {
		err := extract(strings.NewReader(`{ "statusCode": 500, "Internal Server Error"}`))
		assert.True(t, strings.Index(err.Error(), "500") > 0)
		assert.True(t, strings.Index(err.Error(), "Internal Server Error") > 0)
	})

	t.Run("proxy or non json response", func(t *testing.T) {
		err := extract(strings.NewReader("Bad Request"))
		assert.True(t, strings.Index(err.Error(), "Bad Request") > 0)
	})

	t.Run("Fleet generated errors", func(t *testing.T) {
		err := extract(strings.NewReader(`{"statusCode":400,"error":"Bad Request","message":"child \"metadata\" fails because [\"cal\" is not allowed]","validation":{"source":"payload","keys":["metadata.cal"]}}`))
		assert.True(t, strings.Index(err.Error(), "400") > 0)
		assert.True(t, strings.Index(err.Error(), "Bad Request") > 0)
		assert.True(t, strings.Index(err.Error(), "fails because") > 0)
	})
}
