// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows && !requirefips

package cel

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
)

type countingListener struct {
	net.Listener
	accepted *atomic.Int32
}

func (l countingListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err == nil {
		l.accepted.Add(1)
	}
	return c, err
}

func TestAzureCloudConnectorsTokenClientDoesNotUseResourceSocket(t *testing.T) {
	setAzureCloudConnectorsTokenFile(t)

	// The socket path must stay short enough for the platform's sun_path limit.
	dir, err := os.MkdirTemp("", "cel")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "s")
	l, err := (&net.ListenConfig{}).Listen(t.Context(), "unix", socket)
	require.NoError(t, err)
	var socketConns atomic.Int32
	api := &http.Server{Handler: http.NotFoundHandler(), ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = api.Serve(countingListener{Listener: l, accepted: &socketConns}) }()
	t.Cleanup(func() { api.Close() })

	var (
		mu           sync.Mutex
		proxiedHosts []string
	)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		proxiedHosts = append(proxiedHosts, r.Host)
		mu.Unlock()
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(proxy.Close)

	cfg := conf.MustNewConfigFrom(map[string]any{
		"resource.url":                "http+unix://" + socket,
		"resource.proxy_url":          proxy.URL,
		"resource.retry.max_attempts": 1,
		"auth.oauth2": map[string]any{
			"provider":                   "azure",
			"azure.use_cloud_connectors": true,
			"azure.tenant_id":            testAzureTenantID,
			"azure.resource":             "https://management.azure.com/",
			"client.id":                  testAzureClientID,
		},
	})
	c := defaultConfig()
	c.Program = "{}"
	c.Redact = &redact{}
	require.NoError(t, cfg.Unpack(&c))

	log := logptest.NewTestingLogger(t, "")
	client, _, _, _, err := newClient(t.Context(), c, log, nil, v2.Context{Logger: log, ID: "test"}, sdktrace.NewTracerProvider())
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, c.Resource.URL.String(), nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	require.Error(t, err, "the proxy refuses the Entra connection")

	mu.Lock()
	defer mu.Unlock()
	assert.Contains(t, proxiedHosts, "login.microsoftonline.com:443", "the Entra request must use the network transport")
	assert.Zero(t, socketConns.Load(), "the Entra request must not be sent to the resource socket")
}
