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
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"

	"github.com/pkg/errors"
)

// ErrCAPinMissmatch is returned when no pin is matched in the verified chain.
var ErrCAPinMissmatch = errors.New("provided CA certificate pins doesn't match any of the certificate authorities used to validate the certificate")

// verifyPeerCertFunc is a callback defined on the tls.Config struct that will called when a
// TLS connection is used.
type verifyPeerCertFunc func([][]byte, [][]*x509.Certificate) error

// MakeCAPinCallback loops through the verified chains and will try to match the certificates pin.
//
// NOTE: Defining a PIN to check certificates is not a replacement for the normal TLS validations it's
// an additional validation. In fact if you set `InsecureSkipVerify` to true and a PIN, the
// verifiedChains variable will be empty and the added validation will fail.
func MakeCAPinCallback(hashes []string) func([][]byte, [][]*x509.Certificate) error {
	return func(_ [][]byte, verifiedChains [][]*x509.Certificate) error {
		// The chain of trust has been already established before the call to the VerifyPeerCertificate
		// function, after we go through the chain to make sure we have at least a certificate certificate
		//	that match the provided pin.
		for _, chain := range verifiedChains {
			for _, certificate := range chain {
				h := Fingerprint(certificate)
				if matches(hashes, h) {
					return nil
				}
			}
		}

		return ErrCAPinMissmatch
	}
}

// Fingerprint takes a certificate and create a hash of the DER encoded public key.
func Fingerprint(certificate *x509.Certificate) string {
	hash := sha256.Sum256(certificate.RawSubjectPublicKeyInfo)
	return base64.StdEncoding.EncodeToString(hash[:])
}

func matches(pins []string, candidate string) bool {
	for _, pin := range pins {
		if pin == candidate {
			return true
		}
	}
	return false
}
