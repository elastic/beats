// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package lumberjack

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
	client "github.com/elastic/go-lumber/client/v2"
)

const testTimeout = 10 * time.Second

func TestServer(t *testing.T) {
	makeTestConfig := func() config {
		var c config
		c.InitDefaults()
		c.ListenAddress = "localhost:0"
		c.MaxConnections = 1
		c.Keepalive = time.Second
		c.Timeout = time.Second
		return c
	}

	t.Run("empty_batch", func(t *testing.T) {
		testSendReceive(t, makeTestConfig(), 0, nil)
	})

	t.Run("no tls", func(t *testing.T) {
		testSendReceive(t, makeTestConfig(), 10, nil)
	})

	t.Run("tls", func(t *testing.T) {
		clientConf, serverConf := tlsSetup(t)
		clientConf.Certificates = nil

		c := makeTestConfig()
		c.TLS = serverConf
		// Disable mTLS requirements in the server.
		var clientAuth = tlscommon.TLSClientAuthNone
		c.TLS.ClientAuth = &clientAuth
		c.TLS.VerificationMode = tlscommon.VerifyNone

		testSendReceive(t, c, 10, clientConf)
	})

	t.Run("mutual tls", func(t *testing.T) {
		clientConf, serverConf := tlsSetup(t)

		c := makeTestConfig()
		c.TLS = serverConf

		testSendReceive(t, c, 10, clientConf)
	})
}

func testSendReceive(t testing.TB, c config, numberOfEvents int, clientTLSConfig *tls.Config) {
	logp.TestingSetup()
	log := logp.NewLogger(inputName).With("test_name", t.Name())

	ctx, shutdown := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(shutdown)
	collect := newEventCollector(ctx, numberOfEvents)

	// Start server.
	s, err := newServer(c, log, collect.Publish, nil)
	require.NoError(t, err)
	go func() {
		<-ctx.Done()
		s.Close()
	}()

	// Asynchronously send and receive events.
	var wg errgroup.Group
	wg.Go(s.Run)
	wg.Go(func() error {
		// The client returns on error or after an E2E ACK is received.
		// In both cases the test should shutdown.
		defer shutdown()

		return sendData(ctx, t, s.bindAddress, numberOfEvents, clientTLSConfig)
	})

	// Wait for the expected number of events.
	collect.Await(t)

	// Check for errors from client and server.
	require.NoError(t, wg.Wait())
}

func sendData(ctx context.Context, t testing.TB, bindAddress string, numberOfEvents int, clientTLSConfig *tls.Config) error {
	_, port, err := net.SplitHostPort(bindAddress)
	if err != nil {
		return err
	}

	dialFunc := net.Dial
	if clientTLSConfig != nil {
		dialer := &tls.Dialer{
			Config: clientTLSConfig,
		}
		dialFunc = dialer.Dial
	}

	c, err := client.SyncDialWith(dialFunc, net.JoinHostPort("localhost", port))
	if err != nil {
		return fmt.Errorf("client dial error: %w", err)
	}
	defer c.Close()
	go func() {
		<-ctx.Done()
		c.Close()
	}()
	t.Log("Lumberjack client connected.")

	events := make([]interface{}, 0, numberOfEvents)
	for i := 0; i < numberOfEvents; i++ {
		events = append(events, map[string]interface{}{
			"message": "hello world!",
			"index":   i,
		})
	}

	if _, err = c.Send(events); err != nil {
		return fmt.Errorf("failed sending lumberjack events: %w", err)
	}
	t.Log("Lumberjack client sent", len(events), "events.")

	return nil
}

type eventCollector struct {
	sync.Mutex
	events       []beat.Event
	awaitCtx     context.Context // awaitCtx is cancelled when events length is expectedSize.
	awaitCancel  context.CancelFunc
	expectedSize int
}

func newEventCollector(ctx context.Context, expectedSize int) *eventCollector {
	ctx, cancel := context.WithCancel(ctx)
	if expectedSize == 0 {
		cancel()
	}

	return &eventCollector{
		awaitCtx:     ctx,
		awaitCancel:  cancel,
		expectedSize: expectedSize,
	}
}

func (c *eventCollector) Publish(evt beat.Event) {
	c.Lock()
	defer c.Unlock()

	c.events = append(c.events, evt)
	evt.Private.(*batchACKTracker).ACK()

	if len(c.events) == c.expectedSize {
		c.awaitCancel()
	}
}

func (c *eventCollector) Await(t testing.TB) []beat.Event {
	t.Helper()

	<-c.awaitCtx.Done()
	if errors.Is(c.awaitCtx.Err(), context.DeadlineExceeded) {
		t.Fatal(c.awaitCtx.Err())
	}

	c.Lock()
	defer c.Unlock()

	if len(c.events) > c.expectedSize {
		t.Fatalf("more events received than expected, got %d, want %d", len(c.events), c.expectedSize)
	}

	events := make([]beat.Event, len(c.events))
	copy(events, c.events)
	return events
}

var (
	certDataOnce sync.Once
	certData     = struct {
		ca, client, server Cert
	}{}
)

// tlsSetup return client and server configurations ready to test mutual TLS.
func tlsSetup(t *testing.T) (clientConfig *tls.Config, serverConfig *tlscommon.ServerConfig) {
	t.Helper()

	certDataOnce.Do(func() {
		certData.ca, certData.client, certData.server = generateCertData(t)
	})

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(certData.ca.CertPEM(t))

	clientConfig = &tls.Config{
		RootCAs:      certPool,
		Certificates: []tls.Certificate{certData.client.TLSCertificate(t)},
		MinVersion:   tls.VersionTLS12,
	}

	var clientAuth = tlscommon.TLSClientAuthRequired

	serverConfig = &tlscommon.ServerConfig{
		// NOTE: VerifyCertificate is ineffective unless ClientAuth is set to RequireAndVerifyClientCert.
		VerificationMode: tlscommon.VerifyCertificate,
		ClientAuth:       &clientAuth, // tls.RequireAndVerifyClientCert
		CAs: []string{
			string(certData.ca.CertPEM(t)),
		},
		Certificate: tlscommon.CertificateConfig{
			Certificate: string(certData.server.CertPEM(t)),
			Key:         string(certData.server.KeyPEM(t)),
		},
	}

	return clientConfig, serverConfig
}
