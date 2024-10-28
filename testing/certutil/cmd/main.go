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

//nolint:errorlint,forbidigo // it's a cli application
package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
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
	var caPath, caKeyPath, dest, name, names, ipList, prefix, pass string
	var client, rsaflag, noip bool
	flag.StringVar(&caPath, "ca", "",
		"File path for CA in PEM format")
	flag.StringVar(&caKeyPath, "ca-key", "",
		"File path for the CA key in PEM format")
	flag.BoolVar(&rsaflag, "rsa", false,
		"generate a RSA with a 2048-bit key certificate")
	flag.BoolVar(&client, "client", false,
		"generates a client certificate without any IP or SAN/DNS")
	flag.StringVar(&name, "name", "localhost",
		"a single \"Subject Alternate Name values\" for the child certificate. It's added to 'names' if set")
	flag.StringVar(&names, "names", "",
		"a comma separated list of \"Subject Alternate Name values\" for the child certificate")
	flag.BoolVar(&noip, "noip", false,
		"generate a certificate with no IP. It overrides -ips.")
	flag.StringVar(&ipList, "ips", "127.0.0.1",
		"a comma separated list of IP addresses for the child certificate")
	flag.StringVar(&prefix, "prefix", "current timestamp",
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
	if prefix == "current timestamp" {
		prefix = fmt.Sprintf("%d", time.Now().Unix())
	}
	filePrefix := prefix + "-"

	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("error getting current working directory: %v\n", err)
	}
	fmt.Println("files will be witten to:", wd)

	var netIPs []net.IP
	if !noip {
		ips := strings.Split(ipList, ",")
		for _, ip := range ips {
			netIPs = append(netIPs, net.ParseIP(ip))
		}
	}

	var dnsNames []string
	if names != "" {
		dnsNames = strings.Split(names, ",")
	}

	rootCert, rootKey := getCA(rsaflag, caPath, caKeyPath, dest, prefix)
	priv, pub := generateKey(rsaflag)

	childCert, childPair, err := certutil.GenerateGenericChildCert(
		name,
		netIPs,
		priv,
		pub,
		rootKey,
		rootCert,
		certutil.WithCNPrefix(prefix),
		certutil.WithDNSNames(dnsNames...),
		certutil.WithClientCert(client))
	if err != nil {
		panic(fmt.Errorf("error generating child certificate: %w", err))
	}

	if client {
		name = "client"
	}
	savePair(dest, filePrefix+name, childPair)

	if pass != "" {
		fmt.Printf("passphrase present, encrypting \"%s\" certificate key\n",
			name)
		err = os.WriteFile(filePrefix+name+"-passphrase", []byte(pass), 0o600)
		if err != nil {
			panic(fmt.Errorf("error writing passphrase file: %w", err))
		}

		certKeyEnc, err := certutil.EncryptKey(childCert.PrivateKey, pass)
		if err != nil {
			panic(err)
		}

		err = os.WriteFile(filepath.Join(dest, filePrefix+name+"_enc-key.pem"), certKeyEnc, 0o600)
		if err != nil {
			panic(fmt.Errorf("could not save %s certificate encrypted key: %w", filePrefix+name+"_enc-key.pem", err))
		}
	}
}

func generateKey(useRSA bool) (crypto.PrivateKey, crypto.PublicKey) {
	if useRSA {
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(fmt.Errorf("failed to generate RSA key: %v", err))
		}

		return priv, &priv.PublicKey
	}

	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		panic(fmt.Errorf("failed to generate EC key: %v", err))
	}

	return priv, &priv.PublicKey
}

func getCA(rsa bool, caPath, caKeyPath, dest, prefix string) (*x509.Certificate, crypto.PrivateKey) {
	var rootCert *x509.Certificate
	var rootKey crypto.PrivateKey
	var err error

	if caPath == "" && caKeyPath == "" {
		caFn := certutil.NewRootCA
		if rsa {
			caFn = certutil.NewRSARootCA
		}

		var pair certutil.Pair
		rootKey, rootCert, pair, err = caFn(certutil.WithCNPrefix(prefix))
		if err != nil {
			panic(fmt.Errorf("could not create root CA certificate: %w", err))
		}

		savePair(dest, prefix+"-ca", pair)
	} else {
		rootKey, rootCert = loadCA(caPath, caKeyPath)
	}

	return rootCert, rootKey
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
