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

//go:build requirefips

package eslegclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"fmt"
	cfg "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
	"github.com/stretchr/testify/require"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// TestConnectionTLS tries to connect to an Elasticsearch cluster
// (a test HTTPS server) that presents TLS options that are not
// FIPS-compliant. The client, being FIPS-capable, is expected to
// fail the TLS handshake.
func TestConnectionTLS(t *testing.T) {
	server, _ := startTLSServer(t)
	defer server.Close()

	transportSettings := `
ssl:
  enabled: true
`

	var transport httpcommon.HTTPTransportSettings
	err := transport.Unpack(cfg.MustNewConfigFrom(transportSettings))
	require.NoError(t, err)

	transport.TLS.CAs = []string{string(caCertPEM)}

	conn, err := NewConnection(ConnectionSettings{
		URL:       server.URL,
		Transport: transport,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = conn.Connect(ctx)
	require.NoError(t, err)
	// TODO: assert that error is returned and it's related
	// to FIPS-incompatible TLS handshake
}

//go:embed testdata/ca.crt
var caCertPEM []byte

////go:embed testdata/server.crt
//var serverCertPEM []byte

////go:embed testdata/server.key
//var serverKeyPEM []byte // RSA key with length = 2048 bits

//go:embed testdata/fips_invalid.key
var serverKeyPEM []byte // RSA key with length = 1024 bits

//go:embed testdata/fips_invalid.crt
var serverCertPEM []byte

type serverLog struct {
	log strings.Builder
	mu  sync.Mutex
}

func (s *serverLog) Write(data []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.log.Write(data)
}

func (s *serverLog) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.log.String()
}

func startTLSServer(t *testing.T) (*httptest.Server, *serverLog) {
	// Configure server and start it
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCertPEM)

	// Create HTTPS server
	const successResp = `{"message":"hello"}`
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, successResp)
	}))

	serverCert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	require.NoError(t, err)

	server.TLS = &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.NoClientCert,
	}

	logger := new(serverLog)
	server.Config.ErrorLog = log.New(logger, "", 0)

	server.StartTLS()

	return server, logger
}
