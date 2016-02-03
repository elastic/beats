package logstash

import (
	"crypto/tls"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/outputs"
	"github.com/stretchr/testify/assert"
)

type mockServer struct {
	net.Listener
	timeout   time.Duration
	err       error
	handshake func(net.Conn)
}

func (m *mockServer) Addr() string {
	return m.Listener.Addr().String()
}

func (m *mockServer) accept() net.Conn {
	if m.err != nil {
		return nil
	}

	client, err := m.Listener.Accept()
	m.err = err
	return client
}

func (m *mockServer) await() chan net.Conn {
	c := make(chan net.Conn, 1)
	go func() {
		client := m.accept()
		m.handshake(client)
		c <- client
	}()
	return c
}

func newMockServerTLS(t *testing.T, to time.Duration, cert string) *mockServer {
	tcpListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to generate TCP listener")
	}

	tlsConfig, err := outputs.LoadTLSConfig(&outputs.TLSConfig{
		Certificate:    cert + ".pem",
		CertificateKey: cert + ".key",
	})
	if err != nil {
		t.Fatalf("failed to load certificate")
	}

	listener := tls.NewListener(tcpListener, tlsConfig)

	server := &mockServer{Listener: listener, timeout: to}
	server.handshake = func(client net.Conn) {
		if server.err != nil {
			return
		}

		server.clientDeadline(client, server.timeout)
		if server.err != nil {
			return
		}

		tlsConn, ok := client.(*tls.Conn)
		if !ok {
			server.err = errors.New("no tls connection")
			return
		}

		server.err = tlsConn.Handshake()
	}

	return server
}

func newMockServerTCP(t *testing.T, to time.Duration, cert string) *mockServer {
	tcpListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to generate TCP listener")
	}

	server := &mockServer{Listener: tcpListener, timeout: to}
	server.handshake = func(client net.Conn) {}
	return server
}

type mockServerFactory func(*testing.T, time.Duration, string) *mockServer
type transportFactory func(addr string) (TransportClient, error)

func connectTCP(addr string) (TransportClient, error) {
	return newTCPClient(addr, 0)
}

func connectTLS(certName string) transportFactory {
	return func(addr string) (TransportClient, error) {
		tlsConfig, err := outputs.LoadTLSConfig(&outputs.TLSConfig{
			CAs: []string{certName + ".pem"},
		})
		if err != nil {
			return nil, err
		}

		return newTLSClient(addr, 0, tlsConfig)
	}
}

func (m *mockServer) clientDeadline(client net.Conn, to time.Duration) {
	if m.err == nil {
		m.err = client.SetDeadline(time.Now().Add(to))
	}
}

func TestTransportReconnectsOnConnect(t *testing.T) {
	certName := "ca_test"
	timeout := 2 * time.Second
	genCertsForIPIfMIssing(t, net.IP{127, 0, 0, 1}, certName)

	run := func(makeServer mockServerFactory, makeTransp transportFactory) {
		server := makeServer(t, timeout, certName)
		transp, err := makeTransp(server.Addr())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		await := server.await()
		err = transp.Connect(timeout)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		client := <-await

		// force reconnect
		client = nil
		await = server.await()
		err = transp.Connect(timeout)
		assert.NoError(t, err)
		if err != nil {
			client = <-await
			client.Close()
		}

		transp.Close()
	}

	run(newMockServerTCP, connectTCP)
	run(newMockServerTLS, connectTLS(certName))
}

func TestTransportFailConnectUnknownAddress(t *testing.T) {
	certName := "ca_test"
	timeout := 100 * time.Millisecond
	genCertsForIPIfMIssing(t, net.IP{127, 0, 0, 1}, certName)

	invalidAddr := "invalid.dns.fqdn-unknown.invalid:100"

	run := func(makeTransp transportFactory) {
		transp, err := makeTransp(invalidAddr)
		if err != nil {
			t.Fatalf("failed to generate transport client: %v", err)
		}

		err = transp.Connect(timeout)
		assert.NotNil(t, err)
	}

	run(connectTCP)
	run(connectTLS(certName))
}

func TestTransportClosedOnWriteReadError(t *testing.T) {
	certName := "ca_test"
	timeout := 2 * time.Second
	genCertsForIPIfMIssing(t, net.IP{127, 0, 0, 1}, certName)

	run := func(makeServer mockServerFactory, makeTransp transportFactory) {
		server := makeServer(t, timeout, certName)
		transp, err := makeTransp(server.Addr())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		await := server.await()
		err = transp.Connect(timeout)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		client := <-await
		client.Close()
		time.Sleep(5000 * time.Millisecond)

		var buf [10]byte
		transp.Write([]byte("test3"))
		_, err = transp.Read(buf[:])
		assert.NotNil(t, err)
		assert.False(t, transp.IsConnected())
	}

	run(newMockServerTCP, connectTCP)
	run(newMockServerTLS, connectTLS(certName))
}
