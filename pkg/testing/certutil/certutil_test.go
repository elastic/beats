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

package certutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCertificates(t *testing.T) {
	ecRootPair, ecChildPair, err := NewRootAndChildCerts()
	require.NoError(t, err, "could not create EC certificates")

	rsaRootPair, rsaChildPair, err := NewRSARootAndChildCerts()
	require.NoError(t, err, "could not create EC certificates")

	tcs := []struct {
		name      string
		rootPair  Pair
		childPair Pair
	}{
		{
			name:      "EC keys",
			rootPair:  ecRootPair,
			childPair: ecChildPair,
		},
		{
			name:      "RSA keys",
			rootPair:  rsaRootPair,
			childPair: rsaChildPair,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			rootBlock, _ := pem.Decode(tc.rootPair.Cert)
			if rootBlock == nil {
				panic("Failed to parse certificate PEM")

			}
			root, err := x509.ParseCertificate(rootBlock.Bytes)
			if err != nil {
				panic("Failed to parse certificate: " + err.Error())
			}

			childBlock, _ := pem.Decode(tc.childPair.Cert)
			if rootBlock == nil {
				panic("Failed to parse certificate PEM")

			}
			child, err := x509.ParseCertificate(childBlock.Bytes)
			if err != nil {
				panic("Failed to parse certificate: " + err.Error())
			}

			caCertPool := x509.NewCertPool()
			caCertPool.AddCert(root)

			opts := x509.VerifyOptions{
				Roots: caCertPool,
			}

			_, err = child.Verify(opts)
			assert.NoError(t, err, "failed to verify child certificate")
		})
	}
}

func TestGenerateGenericChildCert_dns_cannot_be_empty(t *testing.T) {
	rootKey, rootCACert, _, err := NewRootCA()
	require.NoError(t, err, "could not create root CA certificate")

	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err, "failed to generate EC key")

	tlsCert, _, err := GenerateGenericChildCert(
		"",
		[]net.IP{net.ParseIP("127.0.0.1")},
		priv,
		&priv.PublicKey,
		rootKey,
		rootCACert)
	require.NoError(t, err, "failed to generate child certificate")

	for _, dns := range tlsCert.Leaf.DNSNames {
		assert.NotEmpty(t, dns, "DNSNames contains an empty name")
	}
}
