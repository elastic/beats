// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package oteltest

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/elastic-agent-libs/transport/tlscommontest"
	"github.com/elastic/pkcs8"
)

// GetClientCerts creates client certificates, writes them to a file and return the path of certificate and key
// if passphrase is passed, it is used to encrypt the key file
func GetClientCerts(t *testing.T, caCert tls.Certificate, passphrase string) (certificate string, key string) {
	// create client certificates
	clientCerts, err := tlscommontest.GenSignedCert(caCert, x509.KeyUsageCertSign, false, "client", []string{"localhost"}, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	if err != nil {
		t.Fatalf("could not generate certificates: %s", err)
	}

	tempDir := t.TempDir()
	clientCertPath := filepath.Join(tempDir, "client-cert.pem")
	clientKeyPath := filepath.Join(tempDir, "client-key.pem")

	if passphrase != "" {
		clientKey, err := pkcs8.MarshalPrivateKey(clientCerts.PrivateKey, []byte(passphrase), pkcs8.DefaultOpts)
		if err != nil {
			t.Fatalf("could not marshal private key: %v", err)
		}

		if err = os.WriteFile(clientKeyPath, pem.EncodeToMemory(&pem.Block{
			Type:  "ENCRYPTED PRIVATE KEY",
			Bytes: clientKey,
		}), 0400); err != nil {
			t.Fatalf("could not write client key to file")
		}
	} else {
		clientKey, err := x509.MarshalPKCS8PrivateKey(clientCerts.PrivateKey)
		if err != nil {
			t.Fatalf("could not marshal private key: %v", err)
		}
		if err = os.WriteFile(clientKeyPath, pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: clientKey,
		}), 0400); err != nil {
			t.Fatalf("could not write client key to file")
		}
	}

	if err = os.WriteFile(clientCertPath, pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: clientCerts.Leaf.Raw,
	}), 0400); err != nil {
		t.Fatalf("could not write client certificate to file")
	}

	return clientCertPath, clientKeyPath
}
