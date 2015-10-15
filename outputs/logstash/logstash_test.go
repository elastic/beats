package logstash

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/common/streambuf"
	"github.com/elastic/libbeat/outputs"
)

func testEvent() common.MapStr {
	return common.MapStr{
		"timestamp": common.Time(time.Now()),
		"type":      "log",
		"extra":     10,
		"message":   "message",
	}
}

func testLogstashIndex(test string) string {
	return fmt.Sprintf("beat-logstash-int-%v-%d", test, os.Getpid())
}

func newTestLumberjackOutput(
	t *testing.T,
	test string,
	config *outputs.MothershipConfig,
) outputs.BulkOutputer {
	if config == nil {
		config = &outputs.MothershipConfig{
			Enabled: true,
			TLS:     nil,
			Hosts:   []string{getLogstashHost()},
			Index:   testLogstashIndex(test),
		}

	}

	plugin := outputs.FindOutputPlugin("logstash")
	if plugin == nil {
		t.Fatalf("No logstash output plugin found")
	}

	output, err := plugin.NewOutput("test", config, 0)
	if err != nil {
		t.Fatalf("init logstash output plugin failed: %v", err)
	}

	return output.(outputs.BulkOutputer)
}

func sockReadMessage(buf *streambuf.Buffer, in io.Reader) (*message, error) {
	for {
		// try parse message from buffered data
		msg, err := readMessage(buf)
		if msg != nil || (err != nil && err != streambuf.ErrNoMoreBytes) {
			return msg, err
		}

		// read next bytes from socket if incomplete message in buffer
		buffer := make([]byte, 1024)
		n, err := in.Read(buffer)
		if err != nil {
			return nil, err
		}

		buf.Write(buffer[:n])
	}
}

func sockSendACK(out io.Writer, seq uint32) error {
	buf := streambuf.New(nil)
	buf.WriteByte('1')
	buf.WriteByte('A')
	buf.WriteNetUint32(seq)
	_, err := out.Write(buf.Bytes())
	return err
}

// genCertsIfMIssing generates a testing certificate for ip 127.0.0.1 for
// testing if certificate or key is missing. Generated is used for CA,
// client-auth and server-auth. Use only for testing.
func genCertsIfMIssing(
	t *testing.T,
	capem string,
	cakey string,
) error {
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
		IPAddresses: []net.IP{
			net.IP{127, 0, 0, 1},
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

func TestLogstashTCP(t *testing.T) {
	var serverErr error
	var win, data *message

	// create server with randomized port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create test server")
	}

	// start server
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		// server read timeout
		timeout := 5 * time.Second
		buf := streambuf.New(nil)

		client, err := listener.Accept()
		if err != nil {
			t.Logf("failed on accept: %v", err)
			serverErr = err
			return
		}

		if err := client.SetDeadline(time.Now().Add(timeout)); err != nil {
			serverErr = err
			return
		}
		win, err = sockReadMessage(buf, client)
		if err != nil {
			t.Logf("failed on read window size: %v", err)
			serverErr = err
			return
		}

		if err := client.SetDeadline(time.Now().Add(timeout)); err != nil {
			serverErr = err
			return
		}
		data, err = sockReadMessage(buf, client)
		if err != nil {
			t.Logf("failed on read data frame: %v", err)
			serverErr = err
			return
		}

		if err := client.SetDeadline(time.Now().Add(timeout)); err != nil {
			serverErr = err
			return
		}
		err = sockSendACK(client, 1)
		if err != nil {
			t.Logf("failed on read data frame: %v", err)
			serverErr = err
		}
	}()

	// create lumberjack output client
	config := outputs.MothershipConfig{
		TLS:     nil,
		Timeout: 2,
		Hosts:   []string{listener.Addr().String()},
	}
	output := newTestLumberjackOutput(t, "", &config)

	// send event to server
	event := testEvent()
	output.PublishEvent(nil, time.Now(), event)

	wg.Wait()
	listener.Close()

	// validate output
	assert.Nil(t, serverErr)
	assert.NotNil(t, win)
	if data == nil {
		t.Fatalf("No data received")
	}
	assert.Equal(t, 1, len(data.events))
	data = data.events[0]
	assert.Equal(t, 10.0, data.doc["extra"])
	assert.Equal(t, "message", data.doc["line"])
}

func TestLogstashTLS(t *testing.T) {
	pem := "ca_test.pem"
	key := "ca_test.key"

	var serverErr error
	var win, data *message

	genCertsIfMIssing(t, pem, key)

	tcpListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to generate TCP listener")
	}

	// create lumberjack output client
	config := outputs.MothershipConfig{
		TLS: &outputs.TLSConfig{
			Certificate:    pem,
			CertificateKey: key,
			CAs:            []string{pem},
		},
		Timeout: 5,
		Hosts:   []string{tcpListener.Addr().String()},
	}

	tlsConfig, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		tcpListener.Close()
		t.Fatalf("failed to load certificates")
	}

	listener := tls.NewListener(tcpListener, tlsConfig)

	// start server
	var wg sync.WaitGroup
	var wgReady sync.WaitGroup
	wg.Add(1)
	wgReady.Add(1)
	go func() {
		defer wg.Done()

		for i := 0; i < 3; i++ { // try up to 3 failed connection attempts
			// server read timeout
			timeout := 5 * time.Second
			buf := streambuf.New(nil)
			wgReady.Done()
			client, err := listener.Accept()
			if err != nil {
				continue
			}

			tlsConn, ok := client.(*tls.Conn)
			if !ok {
				serverErr = errors.New("no tls connection")
				return
			}

			if err := client.SetDeadline(time.Now().Add(timeout)); err != nil {
				serverErr = err
				return
			}

			err = tlsConn.Handshake()
			if err != nil {
				serverErr = err
				return
			}

			if err := client.SetDeadline(time.Now().Add(timeout)); err != nil {
				serverErr = err
				return
			}
			win, err = sockReadMessage(buf, client)
			if err != nil {
				serverErr = err
				return
			}

			if err := client.SetDeadline(time.Now().Add(timeout)); err != nil {
				serverErr = err
				return
			}
			data, err = sockReadMessage(buf, client)
			if err != nil {
				serverErr = err
				return
			}
			serverErr = sockSendACK(client, 1)

			return
		}
	}()

	// send event to server
	go func() {
		wgReady.Wait()
		output := newTestLumberjackOutput(t, "", &config)

		event := testEvent()
		output.PublishEvent(nil, time.Now(), event)
	}()

	wg.Wait()
	listener.Close()

	// validate output
	assert.Nil(t, serverErr)
	assert.NotNil(t, win)
	assert.NotNil(t, data)
	if data != nil {
		assert.Equal(t, 1, len(data.events))
		data = data.events[0]
		assert.Equal(t, 10.0, data.doc["extra"])
		assert.Equal(t, "message", data.doc["line"])
	}
}
