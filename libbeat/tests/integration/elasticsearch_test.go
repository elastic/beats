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

//go:build integration

package integration

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/elastic/elastic-agent-libs/testing/certutil"
	"github.com/elastic/mock-es/pkg/api"
)

var esCfg = `
mockbeat:
logging:
  level: debug
  selectors:
    - publisher_pipeline_output
    - esclientleg
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.elasticsearch:
  allow_older_versions: true
  hosts:
    - "%s"
  backoff:
    init: 0.1s
    max: 0.2s
`

func TestESOutputRecoversFromNetworkError(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")

	s, esAddr, _, mr := StartMockES(t, ":4242", 0, 0, 0, 0, 0)

	esURL, err := url.Parse(esAddr)
	if err != nil {
		t.Fatalf("cannot parse mockES URL: %s", err)
	}

	mockbeat.WriteConfigFile(fmt.Sprintf(esCfg, esAddr))
	mockbeat.Start()

	// 1. Wait for one _bulk call
	waitForEventToBePublished(t, mr)

	// 2. Stop the mock-es server
	if err := s.Close(); err != nil {
		t.Fatalf("cannot close mock-es server: %s", err)
	}

	// 3. Wait for connection error logs
	mockbeat.WaitLogsContains(
		fmt.Sprintf(`Get \"%s\": dial tcp %s: connect: connection refused`, esAddr, esURL.Host),

		2*time.Second,
		"did not find connection refused error")

	mockbeat.WaitLogsContains(
		fmt.Sprintf("Attempting to reconnect to backoff(elasticsearch(%s)) with 2 reconnect attempt(s)", esAddr),
		2*time.Second,
		"did not find two tries to reconnect")

	// 4. Restart mock-es on the same port
	s, _, _, mr = StartMockES(t, ":4242", 0, 0, 0, 0, 0)

	// 5. Wait for reconnection logs
	mockbeat.WaitLogsContains(
		fmt.Sprintf("Connection to backoff(elasticsearch(%s)) established", esAddr),
		5*time.Second, // There is a backoff, so ensure we wait enough
		"did not find re connection confirmation")

	// 6. Ensure one new call to _bulk is made
	waitForEventToBePublished(t, mr)
	s.Close()
}

func TestCertificateReload(t *testing.T) {
	// Generate a CA and sign both the server cert and two distinct client certs.
	caPrivKey, caCert, caPair, err := certutil.NewRootCA()
	require.NoError(t, err)

	serverTLSCert, _, err := certutil.GenerateChildCert(
		"mock-es", []net.IP{net.IPv4(127, 0, 0, 1)}, caPrivKey, caCert,
	)
	require.NoError(t, err)

	// client-a and client-b are both signed by the same CA so the server accepts
	// both. Their distinct CNs let us assert which cert the beat is presenting.
	_, clientPairA, err := certutil.GenerateChildCert("client-a", nil, caPrivKey, caCert)
	require.NoError(t, err)
	_, clientPairB, err := certutil.GenerateChildCert("client-b", nil, caPrivKey, caCert)
	require.NoError(t, err)

	// mTLS mock-ES: record the first DNS SAN of each connecting client cert
	// and delegate the actual request handling to the ES mock.
	// certutil.GenerateChildCert places `name` in DNSNames, not CommonName.
	var mu sync.Mutex
	var lastClientName string

	uid := uuid.Must(uuid.NewV4())
	rdr := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(rdr))
	esHandler := api.NewAPIHandler(
		uid, t.Name(), provider, time.Now().Add(24*time.Hour),
		0, 0, 0, 0, 0, 0,
	)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
			if sans := r.TLS.PeerCertificates[0].DNSNames; len(sans) > 0 {
				mu.Lock()
				lastClientName = sans[0]
				mu.Unlock()
			}
		}
		esHandler.ServeHTTP(w, r)
	})

	caPool := x509.NewCertPool()
	caPool.AddCert(caCert)
	l, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{*serverTLSCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
	})
	require.NoError(t, err)

	srv := &http.Server{Handler: handler, ReadHeaderTimeout: time.Second}
	go func() {
		if err := srv.Serve(l); !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("mock ES TLS server: %v", err)
		}
	}()
	t.Cleanup(func() { _ = srv.Close() })
	esAddr := "https://" + l.Addr().String()

	// Write PKI material to files that the beat will read.
	tmpDir := t.TempDir()
	caCertPath := filepath.Join(tmpDir, "ca.pem")
	clientCertPath := filepath.Join(tmpDir, "client.crt")
	clientKeyPath := filepath.Join(tmpDir, "client.key")
	require.NoError(t, os.WriteFile(caCertPath, caPair.Cert, 0600))
	require.NoError(t, os.WriteFile(clientCertPath, clientPairA.Cert, 0600))
	require.NoError(t, os.WriteFile(clientKeyPath, clientPairA.Key, 0600))

	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(`
mockbeat:
logging:
  level: debug
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.elasticsearch:
  allow_older_versions: true
  hosts: ["%s"]
  idle_connection_timeout: 50ms
  backoff:
    init: 0.1s
    max: 0.5s
  ssl:
    certificate_authorities: ["%s"]
    certificate: "%s"
    key: "%s"
    certificate_reload:
      enabled: true
      period: 1s
`, esAddr, caCertPath, clientCertPath, clientKeyPath))

	mockbeat.Start()
	waitForEventToBePublished(t, rdr)

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return lastClientName == "client-a"
	}, 10*time.Second, 100*time.Millisecond,
		"beat should present client-a cert initially")

	// Rotate: overwrite the on-disk files with client-b material.
	require.NoError(t, os.WriteFile(clientCertPath, clientPairB.Cert, 0600))
	require.NoError(t, os.WriteFile(clientKeyPath, clientPairB.Key, 0600))

	// The cert reloader checks every 1s on the TLS-handshake hot-path; the
	// 50ms idle connection timeout forces frequent reconnects so the check
	// is triggered quickly after the rotation.
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return lastClientName == "client-b"
	}, 15*time.Second, 200*time.Millisecond,
		"beat should present client-b cert after rotation without restarting")
}

// waitForEventToBePublished waits for at least one event published
// by inspecting the count for `bulk.create.total` in `mr`. Once
// the counter is > 1, waitForEventToBePublished returns. If that
// does not happen within 10min, then the test fails with a call to
// t.Fatal.
func waitForEventToBePublished(t *testing.T, rdr *sdkmetric.ManualReader) {
	t.Helper()

	require.Eventually(t, func() bool {
		rm := metricdata.ResourceMetrics{}
		err := rdr.Collect(context.Background(), &rm)

		if err != nil {
			t.Fatalf("failed to collect metrics: %v", err)
		}

		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name == "bulk.create.total" {
					total := int64(0)
					//nolint:errcheck // It's a test
					for _, dp := range m.Data.(metricdata.Sum[int64]).DataPoints {
						total += dp.Value
					}
					return total >= 1
				}
			}
		}

		return false
	},
		10*time.Second,
		100*time.Millisecond,
		"at least one bulk request must be made")
}
