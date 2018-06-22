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

	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
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

	tlsConfig, err := tlscommon.LoadTLSConfig(&tlscommon.Config{
		Certificate: tlscommon.CertificateConfig{
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
		tlsConfig, err := tlscommon.LoadTLSConfig(&tlscommon.Config{
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

// GenCertForTestingPurpose generates a testing certificate.
// Generated is used for CA, client-auth and server-auth. Use only for testing.
func GenCertForTestingPurpose(t *testing.T, host, name, keyPassword string) error {
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
	//Could be ip or dns format
	ip := net.ParseIP(host)
	if ip != nil {
		caTemplate.IPAddresses = []net.IP{ip}
	} else {
		caTemplate.DNSNames = []string{host}
	}

	pemBlock, err := genPrivatePem(4096, keyPassword)
	if err != nil {
		t.Fatalf("failed to generate ca private key: %v", err)
	}

	// write key file
	var keyOut *os.File
	keyOut, err = os.OpenFile(cakey, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		t.Fatalf("failed to open key file for writing: %v", err)
	}
	pem.Encode(keyOut, pemBlock)
	keyOut.Close()

	//Decrypt pem block to add it later to the certificate
	if x509.IsEncryptedPEMBlock(pemBlock) {
		pemBlock.Bytes, err = x509.DecryptPEMBlock(pemBlock, []byte(keyPassword))
		if err != nil {
			t.Fatalf("failed to decrypt private key: %v", err)
		}
	}

	var priv *rsa.PrivateKey
	priv, err = x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
	if err != nil {
		t.Fatalf("failed to parse pemBlock to private key: %v", err)
		return err
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

	// write certificate
	certOut, err := os.Create(capem)
	if err != nil {
		t.Fatalf("failed to open cert.pem for writing: %s", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: caBytes})
	certOut.Close()

	return nil
}

func genPrivatePem(bits int, password string) (*pem.Block, error) {
	//Generate private key
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}

	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	//Encrypt pem block from the given password
	if len(password) > 0 {
		block, err = x509.EncryptPEMBlock(rand.Reader, block.Type, block.Bytes, []byte(password), x509.PEMCipherAES256)
		if err != nil {
			return nil, err
		}
	}

	return block, nil
}
