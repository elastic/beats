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
