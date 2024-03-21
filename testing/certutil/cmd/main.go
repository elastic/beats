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
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/elastic/elastic-agent-libs/testing/certutil"
)

func main() {
	var caPath, caKeyPath, dest, name, ipList string
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
	flag.Parse()

	if caPath == "" && caKeyPath != "" || caPath != "" && caKeyPath == "" {
		flag.Usage()
		fmt.Fprintf(flag.CommandLine.Output(),
			"Both 'ca' and 'ca-key' must be specified, or neither should be provided.\nGot ca: %s, ca-key: %s\n",
			caPath, caKeyPath)

	}

	ips := strings.Split(ipList, ",")
	var netIPs []net.IP
	for _, ip := range ips {
		netIPs = append(netIPs, net.ParseIP(ip))
	}

	var rootCert *x509.Certificate
	var rootKey crypto.PrivateKey
	var err error
	if caPath == "" && caKeyPath == "" {
		var pair certutil.Pair
		rootKey, rootCert, pair, err = certutil.NewRootCA()
		if err != nil {
			panic(fmt.Errorf("could not create root CA certificate: %w", err))
		}

		savePair(dest, "ca", pair)
	} else {
		rootKey, rootCert = loadCA(caPath, caKeyPath)
	}

	_, childPair, err := certutil.GenerateChildCert(name, netIPs, rootKey, rootCert)
	if err != nil {
		panic(fmt.Errorf("error generating child certificate: %w", err))
	}

	savePair(dest, name, childPair)
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
