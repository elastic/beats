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

//go:build windows && !requirefips

package translate_ldap_attribute

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarshalSecChannelBindingsLayout(t *testing.T) {
	app := []byte{1, 2, 3, 4}
	b := marshalSecChannelBindings(app)
	require.Len(t, b, secChannelBindingsHeaderSize+len(app))
	require.Equal(t, uint32(len(app)), leUint32(b[24:28]))
	require.Equal(t, uint32(secChannelBindingsHeaderSize), leUint32(b[28:32]))
	require.Equal(t, app, b[secChannelBindingsHeaderSize:])
}

func leUint32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func TestTLSServerEndpointChannelBindingData(t *testing.T) {
	validPEM := `-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6ePLBZuQ/gBDQTEJfcTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTIwMDExMDAwMDAwMFoXDTIxMDExMDAwMDAwMFow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABFy
xLoLAKQA34McPWWFhjIl6cJ3ItZWF8Ku86CzBOct8z92vDn3t5We+TE1hf0Nope6
tZPW2VPoWfn4+5QTjKqjUzBRMA4GA1UdDwEB/wQEAwIFoDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBQ4+0cL9f6cR6SKjbNf
piACtGdR9DAKBggqhkjOPQQDAgNIADBFAiEAxxsxn4MJB8GdVs+lGIGUZTEzi2qNZ
89uiFZTI0xZCEwCIFcSSOXlCSA0rhzBiUcp7Aswcm4BOxdkHz8LBQbIO5e0
-----END CERTIFICATE-----`
	block, _ := pem.Decode([]byte(validPEM))
	require.NotNil(t, block)
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	cs := tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{cert},
	}
	data, err := tlsServerEndpointChannelBindingData(&cs)
	require.NoError(t, err)
	require.True(t, len(data) > len(tlsServerEndPointPrefix))
	require.Equal(t, tlsServerEndPointPrefix, string(data[:len(tlsServerEndPointPrefix)]))
	hash := data[len(tlsServerEndPointPrefix):]
	expected, err := tlsServerEndpointHash(cert.Raw, cert.SignatureAlgorithm)
	require.NoError(t, err)
	require.Equal(t, expected, hash)
}
