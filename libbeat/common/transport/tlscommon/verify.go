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

// Copyright (c) 2009 The Go Authors. All rights reserved.

// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:

//    * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//    * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//    * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.

// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// This file contains code adapted from golang's crypto/tls/handshake_client.go

package tlscommon

import (
	"crypto/x509"
	"time"

	"github.com/pkg/errors"
)

// verifyCertificateExceptServerName is a TLS Certificate verification utility method that verifies that the provided
// certificate chain is valid and is signed by one of the root CAs in the provided tls.Config. It is intended to be
// as similar as possible to the default verify, but does not verify that the provided certificate matches the
// ServerName in the tls.Config.
func verifyCertificateExceptServerName(
	rawCerts [][]byte,
	c *TLSConfig,
) ([]*x509.Certificate, [][]*x509.Certificate, error) {
	// this is where we're a bit suboptimal, as we have to re-parse the certificates that have been presented
	// during the handshake.
	// the verification code here is taken from verifyServerCertificate in crypto/tls/handshake_client.go:824
	certs := make([]*x509.Certificate, len(rawCerts))
	for i, asn1Data := range rawCerts {
		cert, err := x509.ParseCertificate(asn1Data)
		if err != nil {
			return nil, nil, errors.Wrap(err, "tls: failed to parse certificate from server")
		}
		certs[i] = cert
	}

	var t time.Time
	if c.time != nil {
		t = c.time()
	} else {
		t = time.Now()
	}

	// DNSName omitted in VerifyOptions in order to skip ServerName verification
	opts := x509.VerifyOptions{
		Roots:         c.RootCAs,
		CurrentTime:   t,
		Intermediates: x509.NewCertPool(),
	}

	for _, cert := range certs[1:] {
		opts.Intermediates.AddCert(cert)
	}

	headCert := certs[0]

	// defer to the default verification performed
	chains, err := headCert.Verify(opts)
	return certs, chains, err
}
