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

package oteltest

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"text/template"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap/zaptest"

	"github.com/elastic/pkcs8"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/libbeat/otelbeat/beatconverter"
	"github.com/elastic/elastic-agent-libs/testing/proxytest"
	"github.com/elastic/elastic-agent-libs/transport/tlscommontest"
	mockes "github.com/elastic/mock-es/pkg/api"
	"github.com/elastic/opentelemetry-collector-components/extension/beatsauthextension"
)

// This test package tests ES exporter + beatsauth extension together

var beatsAuthName = "beatsauth"
var esExporterName = "elasticsearch"

// create root certificate
var caCert, _ = tlscommontest.GenCA()

// writes ca_cert to a temp file and returns the path
var caFilePath = func(t *testing.T) string {
	caFilePath := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(caFilePath, pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCert.Leaf.Raw}), 0o777), "error writing ca pem blocks to a file")
	return caFilePath
}

type options struct {
	Host                 string
	CACertificate        string
	ClientCert           string
	ClientKey            string
	CATrustedFingerPrint string
	VerificationMode     string
	ProxyURL             string
}

// tests mutual TLS
func TestMTLS(t *testing.T) {

	// create server certificates
	serverCerts, err := tlscommontest.GenSignedCert(caCert, x509.KeyUsageCertSign, false, "server", []string{"localhost"}, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	if err != nil {
		t.Fatalf("could not generate certificates: %s", err)
	}

	// get client certificates paths
	clientCertificate, clientKey := getClientCerts(t, caCert, "")

	// start test server with given server and root certs
	certPool := x509.NewCertPool()
	certPool.AddCert(caCert.Leaf)

	serverName, metricReader := startTestServer(t, &tls.Config{
		// NOTE: client certificates are not verified  unless ClientAuth is set to RequireAndVerifyClientCert.
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
		Certificates: []tls.Certificate{serverCerts},
		MinVersion:   tls.VersionTLS12,
	})

	inputConfig := `
receivers:
  filebeatreceiver:	
    output:
      elasticsearch:
        hosts: {{ .Host }}
        ssl: 
          enabled: true
          certificate_authorities:
            - {{ .CACertificate }}
          certificate: {{ .ClientCert }}
          key: {{ .ClientKey }}
`

	var otelConfigBuffer bytes.Buffer
	require.NoError(t,
		template.Must(template.New("otelConfig").Parse(inputConfig)).Execute(&otelConfigBuffer,
			options{
				Host:          serverName,
				CACertificate: caFilePath(t),
				ClientCert:    clientCertificate,
				ClientKey:     clientKey,
			}))

	// translate beat to beatreceiver config
	output := getTranslatedConf(t, otelConfigBuffer.Bytes())

	// get new test exporter
	exp := newTestESExporter(t, output)

	// get new beats authenticator
	beatsauth := newAuthenticator(t, beatsauthextension.Config{
		BeatAuthconfig: output.Get("extensions::beatsauth").(map[string]any), //nolint: errcheck // it is a test
	})

	// start extension
	host := extensionsMap{component.NewID(component.MustNewType(beatsAuthName)): beatsauth}
	err = beatsauth.Start(t.Context(), host)
	require.NoError(t, err, "could not start extension")

	// start exporter
	err = exp.Start(t.Context(), host)
	require.NoError(t, err, "could not start exporter")

	// send logs
	require.NoError(t, mustSendLogs(t, exp, getLogRecord(t)), "error sending logs")

	// check if data has reached ES
	assertReceivedLogRecord(t, metricReader)
}

func TestKeyPassPhrase(t *testing.T) {

	// create server certificates
	serverCerts, err := tlscommontest.GenSignedCert(caCert, x509.KeyUsageCertSign, false, "server", []string{"localhost"}, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	if err != nil {
		t.Fatalf("could not generate certificates: %s", err)
	}

	// get client certificates paths with key file encrypted in PKCS#8 format
	clientCertificate, clientKey := getClientCerts(t, caCert, "your-password")

	// start test server with given server and root certs
	certPool := x509.NewCertPool()
	certPool.AddCert(caCert.Leaf)

	serverName, metricReader := startTestServer(t, &tls.Config{
		// NOTE: client certificates are not verified  unless ClientAuth is set to RequireAndVerifyClientCert.
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
		Certificates: []tls.Certificate{serverCerts},
		MinVersion:   tls.VersionTLS12,
	})

	inputConfig := `
receivers:
  filebeatreceiver:	
    output:
      elasticsearch:
        hosts: {{ .Host }}
        ssl: 
          enabled: true
          certificate_authorities:
            - {{ .CACertificate }}
          certificate: {{ .ClientCert }}
          key: {{ .ClientKey }}
          key_passphrase: your-password		  
`

	var otelConfigBuffer bytes.Buffer
	require.NoError(t,
		template.Must(template.New("otelConfig").Parse(inputConfig)).Execute(&otelConfigBuffer,
			options{
				Host:          serverName,
				CACertificate: caFilePath(t),
				ClientCert:    clientCertificate,
				ClientKey:     clientKey,
			}))

	// translate beat to beatreceiver config
	output := getTranslatedConf(t, otelConfigBuffer.Bytes())

	// get new test exporter
	exp := newTestESExporter(t, output)

	// get new beats authenticator
	beatsauth := newAuthenticator(t, beatsauthextension.Config{
		BeatAuthconfig: output.Get("extensions::beatsauth").(map[string]any), //nolint: errcheck // it is a test
	})

	// start extension
	host := extensionsMap{component.NewID(component.MustNewType(beatsAuthName)): beatsauth}
	err = beatsauth.Start(t.Context(), host)
	require.NoError(t, err, "could not start extension")

	// start exporter
	err = exp.Start(t.Context(), host)
	require.NoError(t, err, "could not start exporter")

	// send logs
	require.NoError(t, mustSendLogs(t, exp, getLogRecord(t)), "error sending logs")

	// check if data has reached ES
	assertReceivedLogRecord(t, metricReader)
}

// tests ca_trusted_fingerprint
func TestCATrustedFingerPrint(t *testing.T) {
	// create server certificates
	serverCerts, err := tlscommontest.GenSignedCert(caCert, x509.KeyUsageCertSign, false, "server", []string{"localhost"}, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	if err != nil {
		t.Fatalf("could not generate certificates: %s", err)
	}

	// add ca cert to the certificate chain
	serverCerts.Certificate = append(serverCerts.Certificate, caCert.Certificate...)

	fingerprint := tlscommontest.GetCertFingerprint(caCert.Leaf)

	// start test server with given server and root certs
	serverName, metricReader := startTestServer(t, &tls.Config{
		Certificates: []tls.Certificate{serverCerts},
		MinVersion:   tls.VersionTLS12,
	})

	inputConfig := `
receivers:
  filebeatreceiver:	
    output:
      elasticsearch:
        hosts: {{ .Host }}
        ssl: 
          enabled: true
          ca_trusted_fingerprint: {{ .CATrustedFingerPrint }} 
`

	var otelConfigBuffer bytes.Buffer
	require.NoError(t,
		template.Must(template.New("otelConfig").Parse(inputConfig)).Execute(&otelConfigBuffer,
			options{
				Host:                 serverName,
				CATrustedFingerPrint: fingerprint,
			}))

	// translate beat to beatreceiver config
	output := getTranslatedConf(t, otelConfigBuffer.Bytes())

	// get new test exporter
	exp := newTestESExporter(t, output)

	// get new beats authenticator
	beatsauth := newAuthenticator(t, beatsauthextension.Config{
		BeatAuthconfig: output.Get("extensions::beatsauth").(map[string]any), //nolint: errcheck // it is a test
	})

	// start extension
	host := extensionsMap{component.NewID(component.MustNewType(beatsAuthName)): beatsauth}
	err = beatsauth.Start(t.Context(), host)
	require.NoError(t, err, "could not start extension")

	// start exporter
	err = exp.Start(t.Context(), host)
	require.NoError(t, err, "could not start exporter")

	// send logs
	require.NoError(t, mustSendLogs(t, exp, getLogRecord(t)), "error sending logs")

	// check if data has reached ES
	assertReceivedLogRecord(t, metricReader)
}

// tests verification mode
// The test scenarios are taken from https://github.com/elastic/elastic-agent-libs/blob/main/transport/tlscommon/tls_config_test.go#L495
func TestVerificationMode(t *testing.T) {

	testcases := map[string]struct {
		verificationMode string
		expectingError   bool

		// hostname is used to make connection
		hostname string

		// ignoreCerts do not add the Root CA to the trust chain
		ignoreCerts bool

		// commonName used in the Certificate
		commonName string

		// dnsNames is used as the SNA DNSNames
		dnsNames []string

		// ips is used as the SNA IPAddresses
		ips []net.IP
	}{
		"VerifyFull validates domain": {
			verificationMode: "full",
			hostname:         "localhost",
			dnsNames:         []string{"localhost"},
		},
		"VerifyFull validates IPv4": {
			verificationMode: "full",
			hostname:         "127.0.0.1",
			ips:              []net.IP{net.IPv4(127, 0, 0, 1)},
		},
		"VerifyFull domain mismatch returns error": {
			verificationMode: "full",
			hostname:         "localhost",
			dnsNames:         []string{"example.com"},
			expectingError:   true,
		},
		"VerifyFull IPv4 mismatch returns error": {
			verificationMode: "full",
			hostname:         "127.0.0.1",
			ips:              []net.IP{net.IPv4(10, 0, 0, 1)},
			expectingError:   true,
		},
		"VerifyFull does not return error when SNA is empty and legacy Common Name is used": {
			verificationMode: "full",
			hostname:         "localhost",
			commonName:       "localhost",
			expectingError:   false,
		},
		"VerifyFull does not return error when SNA is empty and legacy Common Name is used with IP address": {
			verificationMode: "full",
			hostname:         "127.0.0.1",
			commonName:       "127.0.0.1",
			expectingError:   false,
		},

		"VerifyStrict validates domain": {
			verificationMode: "strict",
			hostname:         "localhost",
			dnsNames:         []string{"localhost"},
		},
		"VerifyStrict validates IPv4": {
			verificationMode: "strict",
			hostname:         "127.0.0.1",
			ips:              []net.IP{net.IPv4(127, 0, 0, 1)},
		},
		"VerifyStrict domain mismatch returns error": {
			verificationMode: "strict",
			hostname:         "127.0.0.1",
			dnsNames:         []string{"example.com"},
			expectingError:   true,
		},
		"VerifyStrict IPv4 mismatch returns error": {
			verificationMode: "strict",
			hostname:         "127.0.0.1",
			ips:              []net.IP{net.IPv4(10, 0, 0, 1)},
			expectingError:   true,
		},
		"VerifyStrict returns error when SNA is empty and legacy Common Name is used": {
			verificationMode: "strict",
			hostname:         "localhost",
			commonName:       "localhost",
			expectingError:   true,
		},
		"VerifyStrict returns error when SNA is empty and legacy Common Name is used with IP address": {
			verificationMode: "strict",
			hostname:         "127.0.0.1",
			commonName:       "127.0.0.1",
			expectingError:   true,
		},
		"VerifyStrict returns error when SNA is empty": {
			verificationMode: "strict",
			hostname:         "localhost",
			expectingError:   true,
		},

		"VerifyCertificate does not validate domain": {
			verificationMode: "certificate",
			hostname:         "localhost",
			dnsNames:         []string{"example.com"},
		},
		"VerifyCertificate does not validate IPv4": {
			verificationMode: "certificate",
			hostname:         "127.0.0.1",
			dnsNames:         []string{"example.com"}, // I believe it cannot be empty
		},
		"VerifyNone accepts untrusted certificates": {
			verificationMode: "none",
			hostname:         "127.0.0.1",
			ignoreCerts:      true,
		},
	}

	for name, test := range testcases {
		t.Run(name, func(t *testing.T) {

			certs, err := tlscommontest.GenSignedCert(caCert, x509.KeyUsageCertSign, false, test.commonName, test.dnsNames, test.ips, false)
			if err != nil {
				t.Fatalf("could not generate certificates: %s", err)
			}

			// start test server with given server and root certs
			serverName, metricReader := startTestServer(t, &tls.Config{
				Certificates: []tls.Certificate{certs},
				MinVersion:   tls.VersionTLS12,
			})

			u, _ := url.Parse(serverName)
			_, port, _ := net.SplitHostPort(u.Host)

			inputConfig := `
receivers:
  filebeatreceiver:	
    output:
      elasticsearch:
        hosts: {{ .Host }}
        ssl: 
          enabled: true
          certificate_authorities:
            - {{ .CACertificate }}
          verification_mode: {{ .VerificationMode }}
`

			var otelConfigBuffer bytes.Buffer
			require.NoError(t,
				template.Must(template.New("otelConfig").Parse(inputConfig)).Execute(&otelConfigBuffer,
					options{
						Host:             fmt.Sprintf("https://%s:%s", test.hostname, port),
						CACertificate:    caFilePath(t),
						VerificationMode: test.verificationMode,
					}))

			// translate beat to beatreceiver config
			output := getTranslatedConf(t, otelConfigBuffer.Bytes())

			// get an instance of es exporter
			exp := newTestESExporter(t, output)

			authConfig := beatsauthextension.Config{
				BeatAuthconfig: output.Get("extensions::beatsauth").(map[string]any), //nolint: errcheck // it is a test
			}

			if test.ignoreCerts {
				delete(authConfig.BeatAuthconfig["ssl"].(map[string]any), "certificate_authorities") //nolint: errcheck // it is a test
			}

			// get new beats authenticator
			beatsauth := newAuthenticator(t, authConfig)

			// start extension
			host := extensionsMap{component.NewID(component.MustNewType(beatsAuthName)): beatsauth}
			err = beatsauth.Start(t.Context(), host)
			require.NoError(t, err, "could not start extension")

			// start exporter
			err = exp.Start(t.Context(), host)
			require.NoError(t, err, "could not start exporter")

			// send logs
			err = mustSendLogs(t, exp, getLogRecord(t))

			if test.expectingError {
				require.Error(t, err, "expected error got none")
				return
			}

			// check if data has reached ES
			assertReceivedLogRecord(t, metricReader)

		})
	}

}

// TestProxyHTTPS tests proxy_url with http and https proxy server
// It also tests proxy_disable configuration
func TestProxyHTTP(t *testing.T) {

	// caCert cert pool
	certPool := x509.NewCertPool()
	certPool.AddCert(caCert.Leaf)

	// create server certificates
	serverCerts, err := tlscommontest.GenSignedCert(caCert, x509.KeyUsageCertSign, false, "server", []string{"localhost"}, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	if err != nil {
		t.Fatalf("could not generate certificates: %s", err)
	}

	// create proxy certificates
	proxyCerts, err := tlscommontest.GenSignedCert(caCert, x509.KeyUsageCertSign, false, "proxy", []string{"localhost"}, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	if err != nil {
		t.Fatalf("could not generate certificates: %s", err)
	}

	testcases := []struct {
		name                  string
		serverTLSConfig       *tls.Config
		proxyOptions          []proxytest.Option
		inputConfig           string
		expectProxiedRequests bool // if the request should go via proxy server
	}{
		{
			name:            "when http proxy url is set",
			serverTLSConfig: nil,
			proxyOptions: []proxytest.Option{proxytest.WithVerboseLog(),
				proxytest.WithRequestLog("https", t.Logf)},
			inputConfig: `
receivers:
  filebeatreceiver:	
    output:
      elasticsearch:
        hosts: {{ .Host }}
        proxy_url: {{ .ProxyURL }}
`,
			expectProxiedRequests: true,
		},
		{
			name: "when http/s proxy url is set",
			serverTLSConfig: &tls.Config{
				Certificates: []tls.Certificate{serverCerts},
				MinVersion:   tls.VersionTLS12,
			},
			proxyOptions: []proxytest.Option{proxytest.WithVerboseLog(),
				proxytest.WithRequestLog("https", t.Logf),
				// we pass ca cert so that proxy server can hijack the incoming client connection
				// create a proxy client that can trust the server's certificate
				proxytest.WithMITMCA(caCert.PrivateKey, caCert.Leaf),
				proxytest.WithHTTPClient(&http.Client{
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{
							MinVersion: tls.VersionTLS13,
							RootCAs:    certPool,
						},
					},
				}),
				proxytest.WithServerTLSConfig(&tls.Config{
					Certificates: []tls.Certificate{proxyCerts},
					MinVersion:   tls.VersionTLS13,
				})},
			inputConfig: `
receivers:
  filebeatreceiver:	
    output:
      elasticsearch:
        hosts: {{ .Host }}
        proxy_url: {{ .ProxyURL }}
        ssl:
          certificate_authorities:
            - {{ .CACertificate }}
`,
			expectProxiedRequests: true,
		},
		{
			name:            "when proxy disable is set",
			serverTLSConfig: nil,
			proxyOptions: []proxytest.Option{proxytest.WithVerboseLog(),
				proxytest.WithRequestLog("https", t.Logf)},
			inputConfig: `
receivers:
  filebeatreceiver:	
    output:
      elasticsearch:
        hosts: {{ .Host }}
        proxy_url: {{ .ProxyURL }}
        proxy_disable: true
`,
			expectProxiedRequests: false,
		},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {

			serverName, metricReader := startTestServer(t, test.serverTLSConfig)
			proxy := proxytest.New(t, test.proxyOptions...)

			if test.serverTLSConfig != nil {
				require.NoErrorf(t, proxy.StartTLS(), "error starting proxy server")
			} else {
				require.NoErrorf(t, proxy.Start(), "error starting proxy server")
			}

			var otelConfigBuffer bytes.Buffer
			require.NoError(t,
				template.Must(template.New("otelConfig").Parse(test.inputConfig)).Execute(&otelConfigBuffer,
					options{
						Host:          serverName,
						ProxyURL:      proxy.URL,
						CACertificate: caFilePath(t),
					}))

			// translate beat to beatreceiver config
			output := getTranslatedConf(t, otelConfigBuffer.Bytes())

			// get new test exporter
			exp := newTestESExporter(t, output)

			// get new beats authenticator
			beatsauth := newAuthenticator(t, beatsauthextension.Config{
				BeatAuthconfig: output.Get("extensions::beatsauth").(map[string]any), //nolint: errcheck // it is a test
			})

			// start extension
			host := extensionsMap{component.NewID(component.MustNewType(beatsAuthName)): beatsauth}
			err = beatsauth.Start(t.Context(), host)
			require.NoError(t, err, "could not start extension")

			// start exporter
			err = exp.Start(t.Context(), host)
			require.NoError(t, err, "could not start exporter")

			// send logs
			require.NoError(t, mustSendLogs(t, exp, getLogRecord(t)), "error sending logs")

			// check if data has reached ES
			assertReceivedLogRecord(t, metricReader)

			if test.expectProxiedRequests {
				// assert if requests have gone via the proxy
				assert.NotEmpty(t, proxy.ProxiedRequests(), "proxy should have captured at least 1 request")
			} else {
				assert.Empty(t, proxy.ProxiedRequests(), "proxy should have captured at least 1 request")
			}
		})
	}

}

// newAuthenticator returns a new beatsauth extension
func newAuthenticator(t *testing.T, config beatsauthextension.Config) extension.Extension {
	beatsauth := beatsauthextension.NewFactory()

	settings := extensiontest.NewNopSettings(beatsauth.Type())
	var err error

	//use testing logger for debugging purposes
	settings.Logger = zaptest.NewLogger(t)

	extension, err := beatsauth.Create(t.Context(), settings, &config)
	require.NoError(t, err, "could not create extension")

	return extension
}

// newTestESExporterWithAuth returns a test exporter
func newTestESExporter(t *testing.T, conf *confmap.Conf) (ESexporter exporter.Logs) {

	f := elasticsearchexporter.NewFactory()
	cfg := &elasticsearchexporter.Config{
		Mapping: elasticsearchexporter.MappingsSettings{
			// we have to set allowed modes
			// this is set on default config in ES exporter but it is not a public type
			Mode:         "bodymap",
			AllowedModes: []string{"bodymap", "ecs", "none", "otel", "raw"},
		},
	}

	esCfg := conf.Get("exporters::elasticsearch")
	esConf := confmap.NewFromStringMap(esCfg.(map[string]any)) //nolint: errcheck // it is a test

	// unmarshall user config into ES exporter config
	require.NoError(t, esConf.Unmarshal(cfg), "error unmarshalling user config into ES config")

	// validate the config
	require.NoError(t, xconfmap.Validate(cfg))

	settings := exportertest.NewNopSettings(component.MustNewType(esExporterName))

	// use testing logger for debugging
	settings.Logger = zaptest.NewLogger(t)

	exp, err := f.CreateLogs(context.Background(), settings, cfg)
	require.NoError(t, err, "could not create exporter.Logs ")
	return exp
}

type extensionsMap map[component.ID]component.Component

func (m extensionsMap) GetExtensions() map[component.ID]component.Component {
	return m
}

// start MOCK ES with given certificates
func startTestServer(t *testing.T, tlsConifg *tls.Config) (serverURL string, metricReader *sdkmetric.ManualReader) {

	rdr := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(rdr))

	mux := http.NewServeMux()
	mux.Handle("/", mockes.NewAPIHandler(
		uuid.Must(uuid.NewV4()),
		"",
		provider,
		time.Now().Add(time.Hour),
		0,
		0, 0, 0, 0, 0))

	server := httptest.NewUnstartedServer(mux)

	if tlsConifg != nil {
		server.TLS = tlsConifg
		server.StartTLS()
	} else {
		server.Start()
	}

	t.Cleanup(func() { server.Close() })

	return server.URL, rdr
}

// getLogRecord returns a single bodymap encoded log record
func getLogRecord(t *testing.T) plog.Logs {
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecords := scopeLogs.LogRecords()
	logRecord := logRecords.AppendEmpty()
	body := pcommon.NewValueMap()
	m := body.Map()
	m.PutStr("@timestamp", time.Now().UTC().Format("2006-01-02T15:04:05.000Z"))
	m.PutInt("id", 1)
	m.PutStr("key", "value")
	body.CopyTo(logRecord.Body())
	return logs
}

// sends log to given exporter
func mustSendLogs(t *testing.T, exporter exporter.Logs, logs plog.Logs) error {
	logs.MarkReadOnly()
	err := exporter.ConsumeLogs(t.Context(), logs)
	return err
}

// getClientCerts creates client certificates, writes them to a file and return the path of certificate and key
// if passphrase is passed, it is used to encrypt the key file
func getClientCerts(t *testing.T, caCert tls.Certificate, passphrase string) (certificate string, key string) {
	// create client certificates
	clientCerts, err := tlscommontest.GenSignedCert(caCert, x509.KeyUsageCertSign, false, "client", []string{"localhost"}, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	if err != nil {
		t.Fatalf("could not generate certificates: %s", err)
	}

	tempDir := t.TempDir()
	clientCertPath := filepath.Join(tempDir, "client-cert.pem")
	clientKeyPath := filepath.Join(tempDir, "client-key.pem")

	if passphrase != "" {
		clientKey, err := pkcs8.MarshalPrivateKey(clientCerts.PrivateKey, []byte(passphrase), pkcs8.DefaultOpts)
		if err != nil {
			t.Fatalf("could not marshal private key: %v", err)
		}

		if err = os.WriteFile(clientKeyPath, pem.EncodeToMemory(&pem.Block{
			Type:  "ENCRYPTED PRIVATE KEY",
			Bytes: clientKey,
		}), 0o777); err != nil {
			t.Fatalf("could not write client key to file")
		}
	} else {
		clientKey, err := x509.MarshalPKCS8PrivateKey(clientCerts.PrivateKey)
		if err != nil {
			t.Fatalf("could not marshal private key: %v", err)
		}
		if err = os.WriteFile(clientKeyPath, pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: clientKey,
		}), 0o777); err != nil {
			t.Fatalf("could not write client key to file")
		}
	}

	if err = os.WriteFile(clientCertPath, pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: clientCerts.Leaf.Raw,
	}), 0o777); err != nil {
		t.Fatalf("could not write client certificate to file")
	}

	return clientCertPath, clientKeyPath
}

func getTranslatedConf(t *testing.T, input []byte) *confmap.Conf {
	c := beatconverter.Converter{}

	conf, err := confmap.NewRetrievedFromYAML(input)
	require.NoError(t, err, "error retrieving config")
	finalConf, err := conf.AsConf()
	require.NoError(t, err, "error transforming config")

	err = c.Convert(t.Context(), finalConf)
	require.NoError(t, err, "error translating config")

	translatedYAML, err := yaml.Marshal(finalConf.ToStringMap())
	require.NoError(t, err)
	t.Logf("Translated configuration:\n%s", translatedYAML)

	return finalConf
}

// assertReceivedLogRecord takes a metric reader and asserts if an event has been successfully indexed
// to mock-es
func assertReceivedLogRecord(t *testing.T, metricReader *sdkmetric.ManualReader) {
	assert.Eventually(t, func() bool {
		rm := metricdata.ResourceMetrics{}
		if err := metricReader.Collect(context.Background(), &rm); err != nil {
			t.Fatalf("could not collect metrics")
		}

		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				// bulk.create.ok is incremented only when the event is accepted
				if m.Name == "bulk.create.ok" {
					switch d := m.Data.(type) {
					case metricdata.Sum[int64]:
						if len(d.DataPoints) >= 1 {
							return true
						}
					}
				}

			}
		}
		return false
	}, 1*time.Minute, 10*time.Second, "did not receive record")
}
