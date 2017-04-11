// Need for unit and integration tests

package transptest

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/armon/go-socks5"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/outputs/transport"
)

// netSOCKS5Proxy starts a new SOCKS5 proxy server that listens on localhost.
//
// Usage:
//  l, tcpAddr := newSOCKS5Proxy(t)
//  defer l.Close()
func newSOCKS5Proxy(t *testing.T) (net.Listener, transport.ProxyConfig) {
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

	// Listen
	go func() {
		err := server.Serve(l)
		if err != nil {
			t.Log(err)
		}
	}()

	tcpAddr := l.Addr().(*net.TCPAddr)
	config := transport.ProxyConfig{URL: fmt.Sprintf("socks5://%s", tcpAddr.String())}
	return l, config
}

func TestTransportReconnectsOnConnect(t *testing.T) {
	l, config := newSOCKS5Proxy(t)
	defer l.Close()

	certName := "ca_test"
	timeout := 2 * time.Second
	GenCertsForIPIfMIssing(t, net.IP{127, 0, 0, 1}, certName)

	testServer(t, &config, func(t *testing.T, makeServer MockServerFactory, proxy *transport.ProxyConfig) {
		server := makeServer(t, timeout, certName, proxy)
		defer server.Close()

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
	l, config := newSOCKS5Proxy(t)
	defer l.Close()

	certName := "ca_test"
	GenCertsForIPIfMIssing(t, net.IP{127, 0, 0, 1}, certName)

	invalidAddr := "invalid.dns.fqdn-unknown.invalid:100"

	run := func(makeTransp TransportFactory, proxy *transport.ProxyConfig) func(*testing.T) {
		return func(t *testing.T) {
			transp, err := makeTransp(invalidAddr, proxy)
			if err != nil {
				t.Fatalf("failed to generate transport client: %v", err)
			}

			err = transp.Connect()
			assert.NotNil(t, err)
		}
	}

	factoryTests := func(f TransportFactory) func(*testing.T) {
		return func(t *testing.T) {
			t.Run("connect", run(f, nil))
			t.Run("socks5", run(f, &config))
		}
	}

	timeout := 100 * time.Millisecond
	t.Run("tcp", factoryTests(connectTCP(timeout)))
	t.Run("tls", factoryTests(connectTLS(timeout, certName)))
}

func TestTransportClosedOnWriteReadError(t *testing.T) {
	l, config := newSOCKS5Proxy(t)
	defer l.Close()

	certName := "ca_test"
	timeout := 2 * time.Second
	GenCertsForIPIfMIssing(t, net.IP{127, 0, 0, 1}, certName)

	testServer(t, &config, func(t *testing.T, makeServer MockServerFactory, proxy *transport.ProxyConfig) {
		server := makeServer(t, timeout, certName, proxy)
		defer server.Close()

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

func testServer(t *testing.T, config *transport.ProxyConfig, run func(*testing.T, MockServerFactory, *transport.ProxyConfig)) {

	runner := func(f MockServerFactory, c *transport.ProxyConfig) func(t *testing.T) {
		return func(t *testing.T) {
			run(t, f, config)
		}
	}

	factoryTests := func(f MockServerFactory) func(t *testing.T) {
		return func(t *testing.T) {
			t.Run("connect", runner(f, nil))
			t.Run("socks5", runner(f, config))
		}
	}

	t.Run("tcp", factoryTests(NewMockServerTCP))
	t.Run("tls", factoryTests(NewMockServerTLS))
}
