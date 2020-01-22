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

// ErrCAPingMatch is returned when no CA pin is matched in the verified chain.
var ErrCAPingMatch = errors.New("provided CA certificate pins doesn't match any of the certificate authorities used to validate the certificate")

type pins []string

func (p *pins) Matches(candidate string) bool {
	for _, pin := range *p {
		if pin == candidate {
			return true
		}
	}
	return false
}

// verifyPeerCertFunc is a callback defined on the tls.Config struct that will called when a
// TLS connection is used.
type verifyPeerCertFunc func([][]byte, [][]*x509.Certificate) error

// MakeCAPinCallback loops throught the verified chains and make sure we have a match in the CA
// that validate the remote chain.
func MakeCAPinCallback(hashes pins) func([][]byte, [][]*x509.Certificate) error {
	return func(_ [][]byte, verifiedChains [][]*x509.Certificate) error {
		// The chain of trust has been already established before the call to the VerifyPeerCertificate
		// function, after we go through the chain to make sure we have CA certificate that match the provided
		// pin.
		for _, chain := range verifiedChains {
			for _, certificate := range chain {
				h := Fingerprint(certificate)
				if hashes.Matches(h) {
					return nil
				}
			}
		}

		return ErrCAPingMatch
	}
}

// Fingerprint takes a certificate and create a hash of the DER encoded public key.
func Fingerprint(certificate *x509.Certificate) string {
	// uses the DER encoded version of the public key to generate the pin.
	hash := sha256.Sum256(certificate.RawSubjectPublicKeyInfo)
	return base64.StdEncoding.EncodeToString(hash[:])
}
