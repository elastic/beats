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

package tlscommon

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"time"
)

// DiagCerts returns a diagnostics hook callback that will validate if the certifiactes (cert + key, and CAs) present in the config are valid.
func (c *Config) DiagCerts() func() []byte {
	if c == nil {
		return func() []byte {
			return []byte("error: nil tlscommon.Config\n")
		}
	}
	return func() []byte {
		var b bytes.Buffer
		logger := log.New(&b, "tlscommon.Config: ", 0)
		logger.Printf("Start diagnostics %s", time.Now().UTC())
		logger.Printf("verification_mode=%s", c.VerificationMode)
		logger.Printf("ca_trusted_fingerprint=%s", c.CATrustedFingerprint)
		logger.Printf("ca_sha256=%v", c.CASha256)

		diagCertificate(logger, &c.Certificate)
		diagCAs(logger, c.CAs)

		return b.Bytes()
	}
}

// DiagCerts returns a diagnostics hook callback that will validate if the certifiactes (cert + key, and CAs) present in the config are valid.
//
// Implementation is mostly a copy of Config.DiagCerts
func (c *ServerConfig) DiagCerts() func() []byte {
	if c == nil {
		return func() []byte {
			return []byte("error: nil tlscommon.ServerConfig\n")
		}
	}
	return func() []byte {
		var b bytes.Buffer
		logger := log.New(&b, "tlscommon.ServerConfig: ", 0)
		logger.Printf("Start diagnostics %s", time.Now().UTC())
		logger.Printf("verification_mode=%s", c.VerificationMode)
		logger.Printf("client_auth=%s", c.ClientAuth)
		logger.Printf("ca_sha256=%v", c.CASha256)

		diagCertificate(logger, &c.Certificate)
		diagCAs(logger, c.CAs)

		return b.Bytes()
	}
}

// diagCertificate will write diagnostics information about the cert/key to the passed logger.
func diagCertificate(logger *log.Logger, cfg *CertificateConfig) {
	prefix := logger.Prefix()
	defer logger.SetPrefix(prefix)
	logger.SetPrefix(prefix + "CertificateSettings: ")

	logger.Print("checking certificate keypair")
	if cfg == nil {
		logger.Print("certificate keypair is nil.")
		return
	}
	crt, err := LoadCertificate(cfg)
	if err != nil {
		logger.Printf("certificate keypair error: %v", err)
		return
	}
	if crt == nil {
		logger.Print("certificate keypair is nil.")
		return
	}
	logger.Print("certificate keypair OK.")
	for i, p := range crt.Certificate {
		cert, err := x509.ParseCertificate(p)
		if err != nil {
			logger.Printf("cert %d - error loading cert: %v", i, err)
			continue
		}
		logger.Printf("cert %d %s", i, CertDiagString(cert))
	}
}

// diagCAs will write diagnostics information about the passed CAs into logger.
func diagCAs(logger *log.Logger, cas []string) {
	prefix := logger.Prefix()
	defer logger.SetPrefix(prefix)
	logger.SetPrefix(prefix + "CertificateAuthorities: ")

	if len(cas) == 0 {
		logger.Print("certificate_authorities not provided, using system certificates.")
		return
	}
	logger.Print("certificate_authorities provided.")
	i := 0
	for _, ca := range cas {
		certs, err := getCACerts(ca)
		if err != nil {
			logger.Printf("Error handling CA: %v", err)
			continue
		}
		for _, cert := range certs {
			logger.Printf("- cert %d %s", i, CertDiagString(cert))
			i++
		}
	}
}

// CertDiagString returns a diagnostics string describing the passed certificate
func CertDiagString(cert *x509.Certificate) string {
	if cert == nil {
		return ""
	}
	return fmt.Sprintf("\n\tSubject=%s\n\tIssuer=%s\n\tIsCA=%v\n\tBasicConstraintsValid=%v\n\tNotBefore=%s\n\tNotAfter=%s\n\tFingerprint=%s\n\tSAN IP=%v\n\tSAN DNS=%v\n\tSAN URI=%v",
		cert.Subject,
		cert.Issuer,
		cert.IsCA,
		cert.BasicConstraintsValid,
		cert.NotBefore,
		cert.NotAfter,
		Fingerprint(cert),
		cert.IPAddresses,
		cert.DNSNames,
		cert.URIs,
	)
}

func getCACerts(ca string) ([]*x509.Certificate, error) {
	r, err := NewPEMReader(ca)
	if err != nil {
		return nil, fmt.Errorf("unable to read CA: %w", err)
	}
	defer r.Close()
	p, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("unable to read ca %s: %w", r, err)
	}
	certs := make([]*x509.Certificate, 0)
	// Loop below copied from go stdlib crypto/x509.CertPool.AppendCertsFromPem
	for len(p) > 0 {
		var block *pem.Block
		block, p = pem.Decode(p)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			continue
		}
		certBytes := block.Bytes
		cert, err := x509.ParseCertificate(certBytes)
		if err != nil {
			continue
		}
		certs = append(certs, cert)
	}
	return certs, nil
}
