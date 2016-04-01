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

	run := func(makeServer MockServerFactory, proxy *transport.ProxyConfig) {
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
	}

	run(NewMockServerTCP, nil)
	run(NewMockServerTLS, nil)
	run(NewMockServerTCP, &config)
	run(NewMockServerTLS, &config)
}

func TestTransportFailConnectUnknownAddress(t *testing.T) {
	l, config := newSOCKS5Proxy(t)
	defer l.Close()

	certName := "ca_test"
	GenCertsForIPIfMIssing(t, net.IP{127, 0, 0, 1}, certName)

	invalidAddr := "invalid.dns.fqdn-unknown.invalid:100"

	run := func(makeTransp TransportFactory, proxy *transport.ProxyConfig) {
		transp, err := makeTransp(invalidAddr, proxy)
		if err != nil {
			t.Fatalf("failed to generate transport client: %v", err)
		}

		err = transp.Connect()
		assert.NotNil(t, err)
	}

	timeout := 100 * time.Millisecond
	run(connectTCP(timeout), nil)
	run(connectTLS(timeout, certName), nil)
	run(connectTCP(timeout), &config)
	run(connectTLS(timeout, certName), &config)
}

func TestTransportClosedOnWriteReadError(t *testing.T) {
	l, config := newSOCKS5Proxy(t)
	defer l.Close()

	certName := "ca_test"
	timeout := 2 * time.Second
	GenCertsForIPIfMIssing(t, net.IP{127, 0, 0, 1}, certName)

	run := func(makeServer MockServerFactory, proxy *transport.ProxyConfig) {
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
	}

	run(NewMockServerTCP, nil)
	run(NewMockServerTLS, nil)
	run(NewMockServerTCP, &config)
	run(NewMockServerTLS, &config)
}
