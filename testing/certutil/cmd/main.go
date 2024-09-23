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
	"crypto"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/elastic/elastic-agent-libs/testing/certutil"
)

func main() {
	var caPath, caKeyPath, dest, name, ipList, filePrefix, pass string
	flag.StringVar(&caPath, "ca", "",
		"File path for CA in PEM format")
	flag.StringVar(&caKeyPath, "ca-key", "",
		"File path for the CA key in PEM format")
	flag.StringVar(&caKeyPath, "dest", "",
		"Directory to save the generated files")
	flag.StringVar(&name, "name", "localhost",
		"used as \"distinguished name\" and \"Subject Alternate Name values\" for the child certificate")
	flag.StringVar(&ipList, "ips", "127.0.0.1",
		"a comma separated list of IP addresses for the child certificate")
	flag.StringVar(&filePrefix, "prefix", "current timestamp",
		"a prefix to be added to the file name. If not provided a timestamp will be used")
	flag.StringVar(&pass, "pass", "",
		"a passphrase to encrypt the certificate key")
	flag.Parse()

	if caPath == "" && caKeyPath != "" || caPath != "" && caKeyPath == "" {
		flag.Usage()
		fmt.Fprintf(flag.CommandLine.Output(),
			"Both 'ca' and 'ca-key' must be specified, or neither should be provided.\nGot ca: %s, ca-key: %s\n",
			caPath, caKeyPath)

	}
	if filePrefix == "" {
		filePrefix = fmt.Sprintf("%d", time.Now().Unix())
	}
	filePrefix += "-"

	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("error getting current working directory: %v\n", err)
	}
	fmt.Println("files will be witten to:", wd)

	ips := strings.Split(ipList, ",")
	var netIPs []net.IP
	for _, ip := range ips {
		netIPs = append(netIPs, net.ParseIP(ip))
	}

	var rootCert *x509.Certificate
	var rootKey crypto.PrivateKey
	if caPath == "" && caKeyPath == "" {
		var pair certutil.Pair
		rootKey, rootCert, pair, err = certutil.NewRootCA()
		if err != nil {
			panic(fmt.Errorf("could not create root CA certificate: %w", err))
		}

		savePair(dest, filePrefix+"ca", pair)
	} else {
		rootKey, rootCert = loadCA(caPath, caKeyPath)
	}

	childCert, childPair, err := certutil.GenerateChildCert(name, netIPs, rootKey, rootCert)
	if err != nil {
		panic(fmt.Errorf("error generating child certificate: %w", err))
	}

	savePair(dest, filePrefix+name, childPair)

	if pass == "" {
		return
	}

	fmt.Printf("passphrase present, encrypting \"%s\" certificate key\n",
		name)
	err = os.WriteFile(filePrefix+name+"-passphrase", []byte(pass), 0o600)
	if err != nil {
		panic(fmt.Errorf("error writing passphrase file: %w", err))
	}

	key, err := x509.MarshalPKCS8PrivateKey(childCert.PrivateKey)
	if err != nil {
		panic(fmt.Errorf("error getting ecdh.PrivateKey from the child's private key: %w", err))
	}

	encPem, err := x509.EncryptPEMBlock( //nolint:staticcheck // we need to drop support for this, but while we don't, it needs to be tested.
		rand.Reader,
		"EC PRIVATE KEY",
		key,
		[]byte(pass),
		x509.PEMCipherAES128)
	if err != nil {
		panic(fmt.Errorf("failed encrypting agent child certificate key block: %v", err))
	}

	certKeyEnc := pem.EncodeToMemory(encPem)

	err = os.WriteFile(filepath.Join(dest, filePrefix+name+"_enc-key.pem"), certKeyEnc, 0o600)
	if err != nil {
		panic(fmt.Errorf("could not save %s certificate encrypted key: %w", filePrefix+name+"_enc-key.pem", err))
	}
}

func loadCA(caPath string, keyPath string) (crypto.PrivateKey, *x509.Certificate) {
	caBytes, err := os.ReadFile(caPath)
	if err != nil {
		panic(fmt.Errorf("failed reading CA file: %w", err))
	}

	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		panic(fmt.Errorf("failed reading CA key file: %w", err))
	}

	tlsCert, err := tls.X509KeyPair(caBytes, keyBytes)
	if err != nil {
		panic(fmt.Errorf("failed generating TLS key pair: %w", err))
	}

	rootCACert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		panic(fmt.Errorf("could not parse certificate: %w", err))
	}

	return tlsCert.PrivateKey, rootCACert
}

func savePair(dest string, name string, pair certutil.Pair) {
	err := os.WriteFile(filepath.Join(dest, name+".pem"), pair.Cert, 0o600)
	if err != nil {
		panic(fmt.Errorf("could not save %s certificate: %w", name, err))
	}

	err = os.WriteFile(filepath.Join(dest, name+"_key.pem"), pair.Key, 0o600)
	if err != nil {
		panic(fmt.Errorf("could not save %s certificate key: %w", name, err))
	}
}
