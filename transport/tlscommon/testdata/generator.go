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

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"

	"github.com/elastic/pkcs8"
)

// main will generate an encrypted pkcs1 keypair for in tlscommon tests
// A static keypair is used to avoid a test panic when tests are ran with GODEBUG=fips140=only
// usage: go run generator.go
func main() {
	key, cert := makeKeyCertPair(blockTypePKCS1Encrypted, "abcd1234")
	keyFile, err := os.Create("key.pkcs1encrypted.pem")
	if err != nil {
		panic(err)
	}
	_, err = keyFile.WriteString(key)
	if err != nil {
		panic(err)
	}
	err = keyFile.Close()

	certFile, err := os.Create("cert.pkcs1encrypted.pem")
	if err != nil {
		panic(err)
	}
	_, err = certFile.WriteString(cert)
	if err != nil {
		panic(err)
	}
	err = certFile.Close()
}

// Below is copied from tlscommon_test.go
const (
	blockTypePKCS1 int = iota
	blockTypePKCS8
	blockTypePKCS1Encrypted
	blockTypePKCS8Encrypted
)

// Setup key+cert pair for the tests
func makeKeyCertPair(blockType int, password string) (string, string) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	var block *pem.Block
	switch blockType {
	case blockTypePKCS1:
		block = &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		}
	case blockTypePKCS8:
		b, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			panic(err)
		}
		block = &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: b,
		}
	case blockTypePKCS1Encrypted:
		var err error
		block, err = x509.EncryptPEMBlock(rand.Reader, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(key), []byte(password), x509.PEMCipherAES256) //nolint:staticcheck // we need to support encrypted private keys
		if err != nil {
			panic(err)
		}
	case blockTypePKCS8Encrypted:
		//TODO: this uses an elastic implementation of pkcs8 as the stdlib does not support password protected pkcs8
		b, err := pkcs8.MarshalPrivateKey(key, []byte(password), nil)
		if err != nil {
			panic(err)
		}
		block = &pem.Block{
			Type:  "ENCRYPTED PRIVATE KEY",
			Bytes: b,
		}
	}

	keyPem := pem.EncodeToMemory(block)
	tml := x509.Certificate{
		SerialNumber: new(big.Int),
		Subject:      pkix.Name{CommonName: "commonName"},
	}
	cert, err := x509.CreateCertificate(rand.Reader, &tml, &tml, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	certPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})
	return string(keyPem), string(certPem)
}
