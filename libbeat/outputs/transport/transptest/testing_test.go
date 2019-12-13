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

// Need for unit and integration tests

package transptest

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	socks5 "github.com/armon/go-socks5"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/outputs/transport"
)

// netSOCKS5Proxy starts a new SOCKS5 proxy server that listens on localhost.
//
// Usage:
//  l, teardown := newSOCKS5Proxy(t)
//  defer teardown()
func newSOCKS5Proxy(t *testing.T) (net.Listener, func()) {
	// Create a SOCKS5 server
	conf := &socks5.Config{}
	server, err := socks5.New(conf)
	if err != nil {
		t.Fatal(err)
	}

	// Create a local listener
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Listen and serve
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := server.Serve(l)
		if err != nil {
			t.Logf("Server error (%T): %+v", err, err)
		}
	}()

	cleanup := func() {
		defer wg.Wait()
		l.Close()
	}

	return l, cleanup
}

func listenerProxyConfig(l net.Listener) *transport.ProxyConfig {
	if l == nil {
		return nil
	}

	tcpAddr := l.Addr().(*net.TCPAddr)
	return &transport.ProxyConfig{
		URL: fmt.Sprintf("socks5://%s", tcpAddr.String()),
	}
}

func TestTransportReconnectsOnConnect(t *testing.T) {
	certName := "ca_test"
	timeout := 2 * time.Second
	GenCertForTestingPurpose(t, "127.0.0.1", certName, "")

	testServer(t, timeout, certName, func(t *testing.T, server *MockServer) {
		transp, err := server.Transp()
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		await := server.Await()
		err = transp.Connect()
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		client := <-await

		// force reconnect
		client = nil
		await = server.Await()
		err = transp.Connect()
		assert.NoError(t, err)
		if err != nil {
			client = <-await
			client.Close()
		}

		transp.Close()
	})
}

func TestTransportFailConnectUnknownAddress(t *testing.T) {
	timeout := 100 * time.Millisecond
	certName := "ca_test"
	GenCertForTestingPurpose(t, "127.0.0.1", certName, "")

	transports := map[string]TransportFactory{
		"tcp": connectTCP(timeout),
		"tls": connectTLS(timeout, certName),
	}

	modes := map[string]struct {
		withProxy bool
	}{
		"connect": {withProxy: false},
		"socks5":  {withProxy: true},
	}

	const invalidAddr = "invalid.dns.fqdn-unknown.invalid:100"

	for name, factory := range transports {
		t.Run(name, func(t *testing.T) {
			for mode, test := range modes {
				t.Run(mode, func(t *testing.T) {
					var listener net.Listener
					if test.withProxy {
						var teardown func()
						listener, teardown = newSOCKS5Proxy(t)
						defer teardown()
					}

					transp, err := factory(invalidAddr, listenerProxyConfig(listener))
					if err != nil {
						t.Fatalf("failed to generate transport client: %v", err)
					}

					err = transp.Connect()
					assert.NotNil(t, err)
				})
			}
		})
	}
}

func TestTransportClosedOnWriteReadError(t *testing.T) {
	certName := "ca_test"
	timeout := 2 * time.Second
	GenCertForTestingPurpose(t, "127.0.0.1", certName, "")

	testServer(t, timeout, certName, func(t *testing.T, server *MockServer) {
		client, transp, err := server.ConnectPair()
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		client.Close()

		var buf [10]byte
		transp.Write([]byte("test3"))
		_, err = transp.Read(buf[:])
		assert.NotNil(t, err)
	})
}

func testServer(t *testing.T, timeout time.Duration, cert string, fn func(t *testing.T, server *MockServer)) {
	transports := map[string]MockServerFactory{
		"tcp": NewMockServerTCP,
		"tls": NewMockServerTLS,
	}

	modes := map[string]struct {
		withProxy bool
	}{
		"connect": {withProxy: false},
		"socks5":  {withProxy: true},
	}

	for name, factory := range transports {
		t.Run(name, func(t *testing.T) {
			for mode, test := range modes {
				t.Run(mode, func(t *testing.T) {
					var listener net.Listener
					if test.withProxy {
						var teardown func()
						listener, teardown = newSOCKS5Proxy(t)
						defer teardown()
					}

					server := factory(t, timeout, cert, listenerProxyConfig(listener))
					defer server.Close()
					fn(t, server)
				})
			}
		})
	}
}
