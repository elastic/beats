package oteltest

import (
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
	"time"

	"github.com/elastic/elastic-agent-libs/transport/tlscommontest"
	mockes "github.com/elastic/mock-es/pkg/api"
	"github.com/elastic/opentelemetry-collector-components/extension/beatsauthextension"
	"github.com/gofrs/uuid/v5"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// This test package tests ES exporter + beatsauth extension together

var beatsAuthName = "beatsauth"
var esExporterName = "elasticsearch"

// tests mutual TLS
func TestMTLS(t *testing.T) {

	// create root certificate
	caCert, err := tlscommontest.GenCA()
	if err != nil {
		t.Fatalf("could not generate root CA certificate: %s", err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(caCert.Leaf)

	// create server certificates
	serverCerts, err := tlscommontest.GenSignedCert(caCert, x509.KeyUsageCertSign, false, "server", []string{"localhost"}, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	if err != nil {
		t.Fatalf("could not generate certificates: %s", err)
	}

	// get client certificates paths
	clientCertificate, clientKey := getClientCerts(t, caCert)

	// start test server with given server and root certs

	serverName, metricReader := startTestServer(t, &tls.Config{
		// NOTE: client certificates are not verified  unless ClientAuth is set to RequireAndVerifyClientCert.
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
		Certificates: []tls.Certificate{serverCerts},
	})

	// get new test exporter with auth
	exp, _ := newTestESExporterWithAuth(t, serverName)

	// get new beats authenticator
	beatsauth := newAuthenticator(t, beatsauthextension.Config{
		BeatAuthconfig: map[string]any{
			"ssl": map[string]any{
				"enabled": true,
				"certificate_authorities": []string{
					string(
						pem.EncodeToMemory(&pem.Block{
							Type:  "CERTIFICATE",
							Bytes: caCert.Leaf.Raw,
						}))},
				"certificate": clientCertificate,
				"key":         clientKey,
			},
		},
	})

	// start extension
	host := extensionsMap{component.NewID(component.MustNewType(beatsAuthName)): beatsauth}
	err = beatsauth.Start(t.Context(), host)
	require.NoError(t, err, "could not start extension")

	// start exporter
	err = exp.Start(t.Context(), host)
	require.NoError(t, err, "could not start exporter")

	// send logs
	mustSendLogs(t, exp, getLogRecord(t))

	// check if data has reached
	require.Eventually(t, func() bool {
		rm := metricdata.ResourceMetrics{}
		if err := metricReader.Collect(context.Background(), &rm); err != nil {
			t.Fatalf("could not collect metrics")
		}

		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				switch d := m.Data.(type) {
				case metricdata.Sum[int64]:
					if len(d.DataPoints) >= 1 {
						return true
					}
				}
			}
		}
		return false
	}, 1*time.Minute, 10*time.Second, "did not receive record")
}

func TestCATrustedFingerPrint(t *testing.T) {

	// create root certificate
	caCert, err := tlscommontest.GenCA()
	if err != nil {
		t.Fatalf("could not generate root CA certificate: %s", err)
	}

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
	})

	// get new test exporter with authenticator set
	exp, _ := newTestESExporterWithAuth(t, serverName)

	// get new beats authenticator
	beatsauth := newAuthenticator(t, beatsauthextension.Config{
		BeatAuthconfig: map[string]any{
			"ssl": map[string]any{
				"enabled":                true,
				"ca_trusted_fingerprint": fingerprint,
			},
		},
	})

	// start extension
	host := extensionsMap{component.NewID(component.MustNewType(beatsAuthName)): beatsauth}
	err = beatsauth.Start(t.Context(), host)
	require.NoError(t, err, "could not start extension")

	// start exporter
	err = exp.Start(t.Context(), host)
	require.NoError(t, err, "could not start exporter")

	// send logs
	mustSendLogs(t, exp, getLogRecord(t))

	// check if data has reached
	require.Eventually(t, func() bool {
		rm := metricdata.ResourceMetrics{}
		if err := metricReader.Collect(context.Background(), &rm); err != nil {
			t.Fatalf("could not collect metrics")
		}

		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				switch d := m.Data.(type) {
				case metricdata.Sum[int64]:
					if len(d.DataPoints) >= 1 {
						return true
					}
				}
			}
		}
		return false
	}, 1*time.Minute, 10*time.Second, "did not receive record")
}

// The test scenarios are taken from https://github.com/khushijain21/elastic-agent-libs/blob/tlsglobal3/transport/tlscommon/tls_config_test.go#L495
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

	// create root certificate
	caCert, err := tlscommontest.GenCA()
	if err != nil {
		t.Fatalf("could not generate root CA certificate: %s", err)
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
			})

			u, _ := url.Parse(serverName) //nolint: errcheck // this is test
			_, port, _ := net.SplitHostPort(u.Host)

			// get new test exporter with authenticator set
			exp, zapLogs := newTestESExporterWithAuth(t, fmt.Sprintf("https://%s:%s", test.hostname, port))

			authConfig := beatsauthextension.Config{
				BeatAuthconfig: map[string]any{
					"ssl": map[string]any{
						"enabled":           true,
						"verification_mode": test.verificationMode,
						"certificate_authorities": []string{
							string(
								pem.EncodeToMemory(&pem.Block{
									Type:  "CERTIFICATE",
									Bytes: caCert.Leaf.Raw,
								}))},
					},
				},
			}

			if test.ignoreCerts {
				delete(authConfig.BeatAuthconfig["ssl"].(map[string]any), "certificate_authorities")
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
			mustSendLogs(t, exp, getLogRecord(t))

			if test.expectingError {
				require.Eventually(t, func() bool {
					return zapLogs.FilterMessageSnippet("bulk indexer flush error").Len() >= 1
				}, 1*time.Minute, 10*time.Second, "did not receive expected error")
				return
			}

			// check if data has reached
			require.Eventually(t, func() bool {
				rm := metricdata.ResourceMetrics{}
				if err := metricReader.Collect(context.Background(), &rm); err != nil {
					t.Fatalf("could not collect metrics")
				}

				for _, sm := range rm.ScopeMetrics {
					for _, m := range sm.Metrics {
						switch d := m.Data.(type) {
						case metricdata.Sum[int64]:
							if len(d.DataPoints) >= 1 {
								return true
							}
						}
					}
				}
				return false
			}, 1*time.Minute, 10*time.Second, "did not receive record")

		})
	}

}

// newAuthenticator returns a new beatsauth extension
func newAuthenticator(t *testing.T, config beatsauthextension.Config) extension.Extension {
	beatsauth := beatsauthextension.NewFactory()

	settings := extensiontest.NewNopSettings(beatsauth.Type())
	var err error

	// we use development logger for debugging purposes
	logConfig := zap.NewDevelopmentConfig()
	logConfig.DisableStacktrace = true
	devLog, err := logConfig.Build()
	require.NoError(t, err, "could not create logger")

	settings.Logger = devLog
	extension, err := beatsauth.Create(t.Context(), settings, &config)
	if err != nil {
		t.Fatalf("could not create extension: %v", err)
	}

	return extension
}

// newTestESExporterWithAuth returns a test exporter with authenticator set
func newTestESExporterWithAuth(t *testing.T, url string, fns ...func(*elasticsearchexporter.Config)) (ESexporter exporter.Logs, logs *observer.ObservedLogs) {
	testauthID := component.NewID(component.MustNewType(beatsAuthName))

	f := elasticsearchexporter.NewFactory()
	queueConfig := exporterhelper.NewDefaultQueueConfig()
	queueConfig.Batch = configoptional.Some(exporterhelper.BatchConfig{
		Sizer:        exporterhelper.RequestSizerTypeItems,
		MinSize:      0,
		FlushTimeout: 1 * time.Second,
	})

	cfg := &elasticsearchexporter.Config{
		Endpoints: []string{url},
		ClientConfig: confighttp.ClientConfig{
			Auth:        configoptional.Some(configauth.Config{AuthenticatorID: testauthID}),
			Compression: "none",
		},
		Mapping: elasticsearchexporter.MappingsSettings{
			Mode:         "bodymap",
			AllowedModes: []string{"bodymap", "ecs", "none", "otel", "raw"},
		},
		QueueBatchConfig: queueConfig,
	}

	for _, fn := range fns {
		fn(cfg)
	}
	require.NoError(t, xconfmap.Validate(cfg))

	settings := exportertest.NewNopSettings(component.MustNewType(esExporterName))

	// development logger
	logConfig := zap.NewDevelopmentConfig()
	logConfig.DisableStacktrace = true
	devLog, err := logConfig.Build()
	require.NoError(t, err, "could not create logger")

	// capture ES exporter logs
	observed, zapLogs := observer.New(zapcore.DebugLevel)
	core := zapcore.NewTee(devLog.Core(), observed)
	settings.Logger = zap.New(core)

	exp, err := f.CreateLogs(context.Background(), settings, cfg)

	require.NoError(t, err)
	return exp, zapLogs
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
	server.TLS = tlsConifg

	server.StartTLS()
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
func mustSendLogs(t *testing.T, exporter exporter.Logs, logs plog.Logs) {
	logs.MarkReadOnly()
	err := exporter.ConsumeLogs(t.Context(), logs)
	require.NoError(t, err)
}

// getClientCerts creates client certificates, writes them to a file and return the path of certificate and key
func getClientCerts(t *testing.T, caCert tls.Certificate) (certificate string, key string) {
	// create client certificates
	clientCerts, err := tlscommontest.GenSignedCert(caCert, x509.KeyUsageCertSign, false, "client", []string{"localhost"}, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	if err != nil {
		t.Fatalf("could not generate certificates: %s", err)
	}

	clientKey, err := x509.MarshalPKCS8PrivateKey(clientCerts.PrivateKey)
	if err != nil {
		t.Fatalf("could not marshal private key: %v", err)
	}

	tempDir := t.TempDir()
	clientCertPath := filepath.Join(tempDir, "client-cert.pem")
	clientKeyPath := filepath.Join(tempDir, "client-key.pem")

	if err = os.WriteFile(clientCertPath, pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: clientCerts.Leaf.Raw,
	}), 0o777); err != nil {
		t.Fatalf("could not write client certificate to file")
	}

	if err = os.WriteFile(clientKeyPath, pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: clientKey,
	}), 0o777); err != nil {
		t.Fatalf("could not write client key to file")
	}

	return clientCertPath, clientKeyPath
}
