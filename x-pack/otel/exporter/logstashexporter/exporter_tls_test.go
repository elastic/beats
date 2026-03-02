// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstashexporter

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	"github.com/elastic/elastic-agent-libs/transport/tlscommontest"
	v2 "github.com/elastic/go-lumber/server/v2"
)

var caCert, _ = tlscommontest.GenCA()
var invalidCACert, _ = tlscommontest.GenCA()

type serverTLSOption struct {
	ClientAuthType tls.ClientAuthType
	AddCAToLeaf    bool
	CommonName     string
	DNSNames       []string
	IPs            []net.IP
}

// TestLumberjackConnection tests exporter connections with different TLS settings
func TestLumberjackConnection(t *testing.T) {
	const testPassphrase = "do-re-mi-fa-so-la-ti-do" // #nosec G101
	caFile := caFilePath(t)
	invalidCAFile := invalidCAFilePath(t)

	tests := []struct {
		name              string
		hostname          string // hostname used in exporter config
		serverTLSOption   *serverTLSOption
		buildClientConfig func() map[string]interface{}
	}{
		{
			name:            "plain connection",
			hostname:        "localhost",
			serverTLSOption: nil,
			buildClientConfig: func() map[string]interface{} {
				return map[string]interface{}{}
			},
		},
		{
			name:     "TLS verification none",
			hostname: "localhost",
			serverTLSOption: &serverTLSOption{
				ClientAuthType: tls.RequestClientCert,
				CommonName:     "whatever",
				DNSNames:       []string{"whatever"},
			},
			buildClientConfig: func() map[string]interface{} {
				return map[string]interface{}{
					"ssl.certificate_authorities": []string{invalidCAFile},
					"ssl.verification_mode":       "none",
				}
			},
		},
		{
			name:     "TLS verification certificate",
			hostname: "localhost",
			serverTLSOption: &serverTLSOption{
				ClientAuthType: tls.RequestClientCert,
				CommonName:     "dont-care",
				DNSNames:       []string{"whatever"},
			},
			buildClientConfig: func() map[string]interface{} {
				return map[string]interface{}{
					"ssl.certificate_authorities": []string{caFile},
					"ssl.verification_mode":       "certificate",
				}
			},
		},
		{
			name:     "TLS verification full",
			hostname: "localhost",
			serverTLSOption: &serverTLSOption{
				ClientAuthType: tls.RequestClientCert,
				CommonName:     "localhost",
			},
			buildClientConfig: func() map[string]interface{} {
				return map[string]interface{}{
					"ssl.certificate_authorities": []string{caFile},
					"ssl.verification_mode":       "full",
				}
			},
		},
		{
			name:     "TLS verification strict",
			hostname: "localhost",
			serverTLSOption: &serverTLSOption{
				ClientAuthType: tls.RequestClientCert,
				DNSNames:       []string{"localhost"},
			},
			buildClientConfig: func() map[string]interface{} {
				return map[string]interface{}{
					"ssl.certificate_authorities": []string{caFile},
					"ssl.verification_mode":       "strict",
				}
			},
		},
		{
			name:     "mutual TLS",
			hostname: "localhost",
			serverTLSOption: &serverTLSOption{
				ClientAuthType: tls.RequireAndVerifyClientCert,
				DNSNames:       []string{"localhost"},
			},
			buildClientConfig: func() map[string]interface{} {
				clientCertificate, clientKey := oteltest.GetClientCerts(t, caCert, "")
				return map[string]interface{}{
					"ssl.certificate_authorities": []string{caFile},
					"ssl.certificate":             clientCertificate,
					"ssl.key":                     clientKey,
				}
			},
		},
		{
			name:     "key passphrase",
			hostname: "localhost",
			serverTLSOption: &serverTLSOption{
				ClientAuthType: tls.RequireAndVerifyClientCert,
				DNSNames:       []string{"localhost"},
			},
			buildClientConfig: func() map[string]interface{} {
				clientCertificate, clientKey := oteltest.GetClientCerts(t, caCert, testPassphrase)
				return map[string]interface{}{
					"ssl.certificate_authorities": []string{caFile},
					"ssl.certificate":             clientCertificate,
					"ssl.key":                     clientKey,
					"ssl.key_passphrase":          testPassphrase,
				}
			},
		},
		{
			name:     "CA trusted fingerprint",
			hostname: "localhost",
			serverTLSOption: &serverTLSOption{
				ClientAuthType: tls.RequestClientCert,
				AddCAToLeaf:    true,
				DNSNames:       []string{"localhost"},
			},
			buildClientConfig: func() map[string]interface{} {
				return map[string]interface{}{
					"ssl.ca_trusted_fingerprint": tlscommontest.GetCertFingerprint(caCert.Leaf),
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverTLSConfig *tls.Config
			if tt.serverTLSOption != nil {
				serverTLSConfig = newServerTLSConfig(t, *tt.serverTLSOption)
			}
			clientConfig := tt.buildClientConfig()
			testWithLumberjackServer(t, tt.hostname, serverTLSConfig, clientConfig)
		})
	}
}

func testWithLumberjackServer(t *testing.T, hostname string, tlsConfig *tls.Config, expConfig map[string]any) {
	server, addr := createLumberjackServer(t, tlsConfig)
	t.Logf("Lumberjack Server address %v", addr)

	// add lumberjack address to exporter config
	var hosts []string
	if v, ok := expConfig["hosts"]; ok {
		hosts, _ = v.([]string)
	}
	hostname = strings.Replace(addr, "127.0.0.1", hostname, 1)
	hosts = append(hosts, hostname)
	expConfig["hosts"] = hosts

	exp := newExporterWithDefaultsWith(t, expConfig)
	t.Cleanup(func() { _ = exp.Shutdown(t.Context()) })
	logs := newTestLogs()

	go func() {
		defer server.Close()

		timeoutCtx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		clientCtx := newTestBeatsClientContext(timeoutCtx)
		err := exp.ConsumeLogs(clientCtx, logs)
		require.NoError(t, err)
	}()

	for batch := range server.ReceiveChan() {
		batch.ACK()

		events := batch.Events
		assert.Len(t, events, 1)
		msg, ok := events[0].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "test log message", msg["value"])
	}
}

func createLumberjackServer(t *testing.T, tlsConfig *tls.Config) (*v2.Server, string) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to generate TCP listener: %v", err)
	}

	if tlsConfig != nil {
		listener = tls.NewListener(listener, tlsConfig)
	}

	server, _ := v2.NewWithListener(listener)
	return server, listener.Addr().String()
}

func newServerTLSConfig(t *testing.T, option serverTLSOption) *tls.Config {
	serverCerts, err := tlscommontest.GenSignedCert(caCert, x509.KeyUsageCertSign, false, option.CommonName, option.DNSNames, option.IPs, false)
	if err != nil {
		t.Fatalf("could not generate certificates: %s", err)
	}
	if option.AddCAToLeaf {
		serverCerts.Certificate = append(serverCerts.Certificate, caCert.Certificate...)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(caCert.Leaf)

	return &tls.Config{
		ClientAuth:   option.ClientAuthType,
		ClientCAs:    certPool,
		Certificates: []tls.Certificate{serverCerts},
		MinVersion:   tls.VersionTLS12,
	}
}

func caFilePath(t *testing.T) string {
	filePath := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(filePath, pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCert.Leaf.Raw}), 0o644), "error writing ca pem blocks to a file")
	return filePath
}
func invalidCAFilePath(t *testing.T) string {
	filePath := filepath.Join(t.TempDir(), "invalid-ca.pem")
	require.NoError(t, os.WriteFile(filePath, pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: invalidCACert.Leaf.Raw}), 0o644), "error writing ca pem blocks to a file")
	return filePath
}
