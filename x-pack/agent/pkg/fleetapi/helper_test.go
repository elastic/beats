// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleetapi

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/kibana"
)

func authHandler(handler http.HandlerFunc, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const key = "Authorization"
		const prefix = "ApiKey "

		v := strings.TrimPrefix(r.Header.Get(key), prefix)
		if v != apiKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		handler(w, r)
	}
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

func withServerWithAuthClient(
	m func(t *testing.T) *http.ServeMux,
	apiKey string,
	test func(t *testing.T, client clienter),
) func(t *testing.T) {

	return withServer(m, func(t *testing.T, host string) {
		log, _ := logger.New()
		cfg := &kibana.Config{
			Host: host,
		}

		client, err := NewAuthWithConfig(log, apiKey, cfg)
		require.NoError(t, err)
		test(t, client)
	})
}
