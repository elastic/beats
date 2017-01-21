package transptest

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type MockServer struct {
	net.Listener
	Timeout   time.Duration
	Err       error
	Handshake func(net.Conn)
	Transp    func() (*transport.Client, error)
}

type MockServerFactory func(*testing.T, time.Duration, string, *transport.ProxyConfig) *MockServer

type TransportFactory func(addr string, proxy *transport.ProxyConfig) (*transport.Client, error)

func (m *MockServer) Addr() string {
	return m.Listener.Addr().String()
}

func (m *MockServer) Accept() net.Conn {
	if m.Err != nil {
		return nil
	}

	client, err := m.Listener.Accept()
	m.Err = err
	return client
}

func (m *MockServer) Await() chan net.Conn {
	c := make(chan net.Conn, 1)
	go func() {
		client := m.Accept()
		m.Handshake(client)
		c <- client
	}()
	return c
}

func (m *MockServer) Connect() (*transport.Client, error) {
	transp, err := m.Transp()
	if err != nil {
		return nil, err
	}

	err = transp.Connect()
	if err != nil {
		return nil, err
	}
	return transp, nil
}

func (m *MockServer) ConnectPair() (net.Conn, *transport.Client, error) {
	transp, err := m.Transp()
	if err != nil {
		return nil, nil, err
	}

	await := m.Await()
	err = transp.Connect()
	if err != nil {
		return nil, nil, err
	}
	client := <-await
	return client, transp, nil
}

func (m *MockServer) ClientDeadline(client net.Conn, to time.Duration) {
	if m.Err == nil {
		m.Err = client.SetDeadline(time.Now().Add(to))
	}
}

func NewMockServerTCP(t *testing.T, to time.Duration, cert string, proxy *transport.ProxyConfig) *MockServer {
	tcpListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to generate TCP listener")
	}

	server := &MockServer{Listener: tcpListener, Timeout: to}
	server.Handshake = func(client net.Conn) {}
	server.Transp = func() (*transport.Client, error) {
		return connectTCP(to)(server.Addr(), proxy)
	}
	return server
}

func NewMockServerTLS(t *testing.T, to time.Duration, cert string, proxy *transport.ProxyConfig) *MockServer {
	tcpListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to generate TCP listener")
	}

	tlsConfig, err := outputs.LoadTLSConfig(&outputs.TLSConfig{
		Certificate: outputs.CertificateConfig{
			Certificate: cert + ".pem",
			Key:         cert + ".key",
		},
	})
	if err != nil {
		t.Fatalf("failed to load certificate")
	}

	listener := tls.NewListener(tcpListener, tlsConfig.BuildModuleConfig(""))

	server := &MockServer{Listener: listener, Timeout: to}
	server.Handshake = func(client net.Conn) {
		if server.Err != nil {
			return
		}

		server.ClientDeadline(client, server.Timeout)
		if server.Err != nil {
			return
		}

		tlsConn, ok := client.(*tls.Conn)
		if !ok {
			server.Err = errors.New("no tls connection")
			return
		}

		server.Err = tlsConn.Handshake()
	}
	server.Transp = func() (*transport.Client, error) {
		return connectTLS(to, cert)(server.Addr(), proxy)
	}

	return server
}

func connectTCP(timeout time.Duration) TransportFactory {
	return func(addr string, proxy *transport.ProxyConfig) (*transport.Client, error) {
		cfg := transport.Config{
			Proxy:   proxy,
			Timeout: timeout,
		}
		return transport.NewClient(&cfg, "tcp", addr, 0)
	}
}

func connectTLS(timeout time.Duration, certName string) TransportFactory {
	return func(addr string, proxy *transport.ProxyConfig) (*transport.Client, error) {
		tlsConfig, err := outputs.LoadTLSConfig(&outputs.TLSConfig{
			CAs: []string{certName + ".pem"},
		})
		if err != nil {
			return nil, err
		}

		cfg := transport.Config{
			Proxy:   proxy,
			TLS:     tlsConfig,
			Timeout: timeout,
		}
		return transport.NewClient(&cfg, "tcp", addr, 0)
	}
}

// genCertsIfMIssing generates a testing certificate for ip 127.0.0.1 for
// testing if certificate or key is missing. Generated is used for CA,
// client-auth and server-auth. Use only for testing.
func GenCertsForIPIfMIssing(
	t *testing.T,
	ip net.IP,
	name string,
) error {
	capem := name + ".pem"
	cakey := name + ".key"

	_, err := os.Stat(capem)
	if err == nil {
		_, err = os.Stat(cakey)
		if err == nil {
			return nil
		}
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		t.Fatalf("failed to generate serial number: %s", err)
	}

	caTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:            []string{"US"},
			Organization:       []string{"elastic"},
			OrganizationalUnit: []string{"beats"},
		},
		Issuer: pkix.Name{
			Country:            []string{"US"},
			Organization:       []string{"elastic"},
			OrganizationalUnit: []string{"beats"},
			Locality:           []string{"locality"},
			Province:           []string{"province"},
			StreetAddress:      []string{"Mainstreet"},
			PostalCode:         []string{"12345"},
			SerialNumber:       "23",
			CommonName:         "*",
		},
		IPAddresses: []net.IP{ip},

		SignatureAlgorithm:    x509.SHA512WithRSA,
		PublicKeyAlgorithm:    x509.ECDSA,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		SubjectKeyId:          []byte("12345"),
		BasicConstraintsValid: true,
		IsCA: true,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth},
		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature |
			x509.KeyUsageCertSign,
	}

	// generate keys
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		t.Fatalf("failed to generate ca private key: %v", err)
	}
	pub := &priv.PublicKey

	// generate certificate
	caBytes, err := x509.CreateCertificate(
		rand.Reader,
		&caTemplate,
		&caTemplate,
		pub, priv)
	if err != nil {
		t.Fatalf("failed to generate ca certificate: %v", err)
	}

	// write key file
	keyOut, err := os.OpenFile(cakey, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		t.Fatalf("failed to open key file for writing: %v", err)
	}
	pem.Encode(keyOut, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()

	// write certificate
	certOut, err := os.Create(capem)
	if err != nil {
		t.Fatalf("failed to open cert.pem for writing: %s", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: caBytes})
	certOut.Close()

	return nil
}
