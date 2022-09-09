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

package tcp

import (
	"crypto/tls"
	"crypto/x509"
	"net"

	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
)

// MetadataCallback returns common metadata about a tcp connection
func MetadataCallback(conn net.Conn) inputsource.NetworkMetadata {
	return inputsource.NetworkMetadata{
		RemoteAddr: conn.RemoteAddr(),
		TLS:        extractSSLInformation(conn),
	}
}

func extractSSLInformation(c net.Conn) *inputsource.TLSMetadata {
	if tls, ok := c.(*tls.Conn); ok {
		state := tls.ConnectionState()
		return &inputsource.TLSMetadata{
			TLSVersion:       tlscommon.ResolveTLSVersion(state.Version),
			CipherSuite:      tlscommon.ResolveCipherSuite(state.CipherSuite),
			ServerName:       state.ServerName,
			PeerCertificates: extractCertificate(state.PeerCertificates),
		}
	}
	return nil
}

func extractCertificate(certificates []*x509.Certificate) []string {
	strCertificate := make([]string, len(certificates))
	for idx, c := range certificates {
		// Ignore errors here, problematics cert have failed
		// the handshake at this point.
		b, _ := x509.MarshalPKIXPublicKey(c.PublicKey)
		strCertificate[idx] = string(b)
	}
	return strCertificate
}
