package lumberjack

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"math/rand"
	"net"
	"time"

	"github.com/elastic/libbeat/logp"
)

// TLSConfig lists certificate and key files to be loaded for TLS based connections.
type TLSConfig struct {
	Certificate string
	Key         string
	CAs         []string
}

// TransportClient interfaces adds (re-)connect support to net.Conn.
type TransportClient interface {
	net.Conn
	Connect(timeout time.Duration) error
	IsConnected() bool
}

type tcpClient struct {
	hostport  string
	connected bool
	net.Conn
}

type tlsClient struct {
	tcpClient
	tls tlsConfig
}

type tlsConfig struct {
	// MinVersion contains the minimum SSL/TLS version that is acceptable.
	// If zero, then TLS 1.0 is taken as the minimum.
	MinVersion uint16

	// Certificates contains one or more certificate chains
	// to present to the other side of the connection.
	// Server configurations must include at least one certificate.
	Certificates []tls.Certificate

	// RootCAs defines the set of root certificate authorities
	// that clients use when verifying server certificates.
	// If RootCAs is nil, TLS uses the host's root CA set.
	RootCAs *x509.CertPool
}

var (
	// ErrNotACertificate indicates a PEM file to be loaded not being a valid
	// PEM file or certificate.
	ErrNotACertificate = errors.New("file is not a certificate")

	// ErrCertificateNoKey indicate a configuration error with missing key file
	ErrCertificateNoKey = errors.New("key file not configured")

	// ErrKeyNoCertificate indicate a configuration error with missing certificate file
	ErrKeyNoCertificate = errors.New("certificate file not configured")
)

func newTCPClient(host string) (*tcpClient, error) {
	return &tcpClient{hostport: host}, nil
}

func (c *tcpClient) Connect(timeout time.Duration) error {
	if c.IsConnected() {
		_ = c.Close()
	}

	host, port, err := net.SplitHostPort(c.hostport)
	if err != nil {
		return err
	}

	// TODO: address lookup copied from logstash-forwarded. Really required?
	addresses, err := net.LookupHost(host)
	c.Conn = nil
	if err != nil {
		logp.Warn("DNS lookup failure \"%s\": %s", host, err)
		return err
	}

	// connect to random address
	// Use randomization on DNS reported addresses combined with timeout and ACKs
	// to spread potential load when starting up large number of beats using
	// lumberjack.
	//
	// RFCs discussing reasons for ignoring order of DNS records:
	// http://www.ietf.org/rfc/rfc3484.txt
	// > is specific to locality-based address selection for multiple dns
	// > records, but exists as prior art in "Choose some different ordering for
	// > the dns records" done by a client
	//
	// https://tools.ietf.org/html/rfc1794
	// > "Clients, of course, may reorder this information" - with respect to
	// > handling order of dns records in a response. address :=
	address := addresses[rand.Int()%len(addresses)]
	addressport := net.JoinHostPort(address, port)
	conn, err := net.DialTimeout("tcp", addressport, timeout)
	if err != nil {
		return err
	}

	c.Conn = conn
	c.connected = true
	return nil
}

func (c *tcpClient) IsConnected() bool {
	return c.connected
}

func (c *tcpClient) Close() error {
	err := c.Conn.Close()
	c.connected = false
	return err
}

func loadTLSConfig(config *TLSConfig) (*tlsConfig, error) {
	var tlsconfig tlsConfig

	// Support minimal TLS 1.0.
	// TODO: check supported JRuby versions for logstash supported
	//       TLS 1.1 and switch
	tlsconfig.MinVersion = tls.VersionTLS10

	hasCertificate := config.Certificate != ""
	hasKey := config.Key != ""
	switch {
	case hasCertificate && !hasKey:
		return nil, ErrCertificateNoKey
	case !hasCertificate && hasKey:
		return nil, ErrKeyNoCertificate
	case hasCertificate && hasKey:
		cert, err := tls.LoadX509KeyPair(config.Certificate, config.Key)
		if err != nil {
			logp.Critical("Failed loading client certificate", err)
			return nil, err
		}
		tlsconfig.Certificates = []tls.Certificate{cert}
	}

	if len(config.CAs) > 0 {
		tlsconfig.RootCAs = x509.NewCertPool()
	}
	for _, caFile := range config.CAs {
		pemData, err := ioutil.ReadFile(caFile)
		if err != nil {
			logp.Critical("Failed reading CA certificate: %s", err)
			return nil, err
		}

		block, _ := pem.Decode(pemData)
		if block == nil {
			logp.Critical("Failed to decode PEM. Is certificate %s valid?", caFile)
			return nil, ErrNotACertificate
		}
		if block.Type != "CERTIFICATE" {
			logp.Critical("PEM File %s is not a certificate", caFile)
			return nil, ErrNotACertificate
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			logp.Critical("Failed to parse certificate file %s", caFile)
			return nil, ErrNotACertificate
		}

		tlsconfig.RootCAs.AddCert(cert)
	}

	return &tlsconfig, nil
}

func newTLSClient(host string, tls tlsConfig) (*tlsClient, error) {
	c := tlsClient{}
	c.hostport = host
	c.tls = tls
	return &c, nil
}

func (c *tlsClient) Connect(timeout time.Duration) error {
	host, _, err := net.SplitHostPort(c.hostport)
	if err != nil {
		return err
	}

	var tlsconfig tls.Config
	tlsconfig.MinVersion = c.tls.MinVersion
	tlsconfig.RootCAs = c.tls.RootCAs
	tlsconfig.Certificates = c.tls.Certificates
	tlsconfig.ServerName = host

	if err := c.tcpClient.Connect(timeout); err != nil {
		return err
	}

	socket := tls.Client(c.Conn, &tlsconfig)
	if err := socket.SetDeadline(time.Now().Add(timeout)); err != nil {
		_ = socket.Close()
		return c.onFail(err)
	}
	if err := socket.Handshake(); err != nil {
		_ = socket.Close()
		return c.onFail(err)
	}

	c.Conn = socket
	c.connected = true
	return nil
}

func (c *tlsClient) onFail(err error) error {
	c.Conn = nil
	c.connected = false
	return err
}
