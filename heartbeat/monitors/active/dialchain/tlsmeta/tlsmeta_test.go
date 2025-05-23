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

package tlsmeta

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/testslike"
	"github.com/elastic/go-lookslike/validator"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/look"
)

// Tests for the non-cert fields
func TestAddTLSMetadata(t *testing.T) {
	// We always test with this one cert because addCertificateMetadata
	// is tested in detail elsewhere
	certs := []*x509.Certificate{parseCert(t, elasticCert)}
	certMetadata := CertFields(certs[0], nil)

	scenarios := []struct {
		name      string
		connState tls.ConnectionState
		duration  time.Duration
		expected  mapstr.M
	}{
		{
			"simple TLSv1.1",
			tls.ConnectionState{
				Version:           tls.VersionTLS11,
				HandshakeComplete: true,
				PeerCertificates:  certs,
				CipherSuite:       tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
				ServerName:        "example.net",
			},
			time.Duration(1),
			mapstr.M{
				"established":      true,
				"rtt":              mapstr.M{"handshake": look.RTT(time.Duration(1))},
				"version_protocol": "tls",
				"version":          "1.1",
				"cipher":           "ECDHE-ECDSA-AES-256-CBC-SHA",
			},
		},
		{
			"TLSv1.2 with next_protocol",
			tls.ConnectionState{
				Version:            tls.VersionTLS12,
				HandshakeComplete:  true,
				PeerCertificates:   certs,
				CipherSuite:        tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
				ServerName:         "example.net",
				NegotiatedProtocol: "h2",
			},
			time.Duration(1),
			mapstr.M{
				"established":      true,
				"rtt":              mapstr.M{"handshake": look.RTT(time.Duration(1))},
				"version_protocol": "tls",
				"version":          "1.2",
				"cipher":           "ECDHE-ECDSA-AES-256-CBC-SHA",
				"next_protocol":    "h2",
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			// Nest under the TLS namespace to match actual output
			expected := mapstr.M{"tls": s.expected}

			// Always add in the cert metadata since we test that in other test funcs, not here
			expected.DeepUpdate(mapstr.M{"tls": certMetadata})

			fields := mapstr.M{}
			AddTLSMetadata(fields, s.connState, s.duration)
			require.Equal(t, expected, fields)
		})
	}
}

func TestCertFields(t *testing.T) {
	cert := parseCert(t, elasticCert)
	esChainCert := parseCert(t, elasticChainCert)

	certNotBefore, err := time.Parse(time.RFC3339, "2019-08-16T01:40:25Z")
	require.NoError(t, err)
	certNotAfter, err := time.Parse(time.RFC3339, "2020-07-16T03:15:39Z")
	require.NoError(t, err)

	elasticCertFields := lookslike.Strict(lookslike.MustCompile(map[string]interface{}{
		"certificate_not_valid_after":  certNotAfter,
		"certificate_not_valid_before": certNotBefore,
		"server": mapstr.M{
			"hash": mapstr.M{
				"sha1":   "b7b4b89ef0d0caf39d223736f0fdbb03c7b426f1",
				"sha256": "12b00d04db0db8caa302bfde043e88f95baceb91e86ac143e93830b4bbec726d",
			},
			"x509": mapstr.M{
				"issuer": mapstr.M{
					"common_name":        "GlobalSign CloudSSL CA - SHA256 - G3",
					"distinguished_name": "CN=GlobalSign CloudSSL CA - SHA256 - G3,O=GlobalSign nv-sa,C=BE",
				},
				"subject": mapstr.M{
					"common_name":        "r2.shared.global.fastly.net",
					"distinguished_name": "CN=r2.shared.global.fastly.net,O=Fastly\\, Inc.,L=San Francisco,ST=California,C=US",
				},
				"not_after":            certNotAfter,
				"not_before":           certNotBefore,
				"serial_number":        "26610543540289562361990401194",
				"signature_algorithm":  "SHA256-RSA",
				"public_key_algorithm": "RSA",
				"public_key_size":      2048,
				"public_key_exponent":  65537,
			},
		},
	}))

	letsEncryptCert := letsEncryptCerts(t)[0]

	letsEncryptCertFields := func(notBefore time.Time, notAfter time.Time) validator.Validator {
		return lookslike.Strict(lookslike.MustCompile(map[string]interface{}{
			"certificate_not_valid_before": notBefore,
			"certificate_not_valid_after":  notAfter,
			"server": mapstr.M{
				"hash": mapstr.M{
					"sha1":   "98d7ca35e3608b0ee7accc9ec665babdbdc6e39c",
					"sha256": "ada85303e669f8ad9c537c2355936a11d95089ff4dbae57db664937f266f61a5",
				},
				"x509": mapstr.M{
					"issuer": mapstr.M{
						"common_name":        letsEncryptCert.Issuer.CommonName,
						"distinguished_name": "CN=R3,O=Let's Encrypt,C=US",
					},
					"subject": mapstr.M{
						"common_name":        letsEncryptCert.Subject.CommonName,
						"distinguished_name": "CN=lencr.org",
					},
					"not_before":           notBefore,
					"not_after":            notAfter,
					"serial_number":        letsEncryptCert.SerialNumber.String(),
					"signature_algorithm":  letsEncryptCert.SignatureAlgorithm.String(),
					"public_key_algorithm": letsEncryptCert.PublicKeyAlgorithm.String(),
					"public_key_curve":     "P-256",
				},
			},
		}))
	}

	leCertNotBefore, err := time.Parse(time.RFC3339, "2022-10-05T01:40:24Z")
	require.NoError(t, err)
	leCertNotAfter, err := time.Parse(time.RFC3339, "2023-01-03T01:40:23Z")
	require.NoError(t, err)

	// This is a bit awkward because our test chain includes two roots.
	// In the real world the go stdlib would split this single chain chains across two verified chains
	// however, the behavior in this scenario is correct
	leXSignCertNotBefore, err := time.Parse(time.RFC3339, "2022-10-05T01:40:24Z")
	require.NoError(t, err)
	leXSignCertNotAfter, err := time.Parse(time.RFC3339, "2021-09-30T14:01:15Z")
	require.NoError(t, err)

	scenarios := []struct {
		name           string
		cert           *x509.Certificate
		chain          [][]*x509.Certificate
		expectedFields validator.Validator
	}{
		{
			"single cert fields should all be present",
			cert,
			nil,
			elasticCertFields,
		},
		{
			"cert chain should still show single cert fields",
			cert,
			[][]*x509.Certificate{{cert, esChainCert}},
			elasticCertFields,
		},
		{
			"cross signed chain, one root included",
			letsEncryptCert,
			[][]*x509.Certificate{letsEncryptCerts(t)},
			letsEncryptCertFields(leCertNotBefore, leCertNotAfter),
		},
		{
			"complex cross signed chain, multiple roots included, at least one expired",
			letsEncryptCert,
			[][]*x509.Certificate{letsEncryptXSignedDSTX3(t)},
			letsEncryptCertFields(leXSignCertNotBefore, leXSignCertNotAfter),
		},
		{
			"reversed cert should expire at same time",
			cert,
			[][]*x509.Certificate{{esChainCert, cert}},
			elasticCertFields,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tls := CertFields(scenario.cert, scenario.chain)
			require.NoError(t, err)
			testslike.Test(t, scenario.expectedFields, tls)
		})
	}
}

// TestCertExpirationMetadata exhaustively tests not before / not after calculation.
func TestCertExpirationMetadata(t *testing.T) {
	goodNotBefore := time.Now().Add(-time.Hour)
	goodNotAfter := time.Now().Add(time.Hour)
	goodCert := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore:             goodNotBefore,
		NotAfter:              goodNotAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	expiredNotAfter := time.Now().Add(-time.Hour)
	expiredCert := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore:             goodNotBefore,
		NotAfter:              expiredNotAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	missingNotBeforeCert := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotAfter:              goodNotAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	missingNotAfterCert := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore:             goodNotBefore,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// notBefore is intentionally not a pointer type because go certificates don't have nullable time types
	// we cheat a bit and make not after nullable because there's no valid reason to create a cert with go's zero
	// time.
	// see the AddCertMetadata function for more info on this.
	type expected struct {
		notBefore time.Time
		notAfter  *time.Time
	}
	tests := []struct {
		name     string
		certs    []*x509.Certificate
		expected expected
	}{
		{
			"Valid cert",
			[]*x509.Certificate{&goodCert},
			expected{
				notBefore: goodNotBefore,
				notAfter:  &goodNotAfter,
			},
		},
		{
			"Expired cert",
			[]*x509.Certificate{&expiredCert},
			expected{
				notBefore: goodNotBefore,
				notAfter:  &expiredNotAfter,
			},
		},
		{
			"Missing not before",
			[]*x509.Certificate{&missingNotBeforeCert},
			expected{
				notAfter: &goodNotAfter,
			},
		},
		{
			"Missing not after",
			[]*x509.Certificate{&missingNotAfterCert},
			expected{
				notBefore: goodNotBefore,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notBefore, notAfter := calculateCertTimestamps(tt.certs)

			require.Equal(t, tt.expected.notBefore, notBefore)
			if tt.expected.notAfter != nil {
				require.Equal(t, tt.expected.notAfter, notAfter)
			} else {
				require.Nil(t, notAfter)
			}
		})
	}
}

func parseCert(t *testing.T, pemStr string) *x509.Certificate {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		require.Fail(t, "Test cert could not be parsed")
		return nil
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	return cert
}

var elasticCert = `-----BEGIN CERTIFICATE-----
MIIPLzCCDhegAwIBAgIMVfu5x96/CYCdEsyqMA0GCSqGSIb3DQEBCwUAMFcxCzAJ
BgNVBAYTAkJFMRkwFwYDVQQKExBHbG9iYWxTaWduIG52LXNhMS0wKwYDVQQDEyRH
bG9iYWxTaWduIENsb3VkU1NMIENBIC0gU0hBMjU2IC0gRzMwHhcNMTkwODE2MDE0
MDI1WhcNMjAwNzE2MDMxNTM5WjB3MQswCQYDVQQGEwJVUzETMBEGA1UECAwKQ2Fs
aWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNjbzEVMBMGA1UECgwMRmFzdGx5
LCBJbmMuMSQwIgYDVQQDDBtyMi5zaGFyZWQuZ2xvYmFsLmZhc3RseS5uZXQwggEi
MA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCnvoHpOqA6CM06MlGViMGMFC4G
YFFEe03GQ5jG3uEUbMNPbl0MSxaWle5xZOVaPcIrV7qyE5yKKDv1fT1e8EkwR+3t
nTK4k2QvH6dPtSPlGHVIjBtS17gM939eZvpvUPxmUc5Ov9cbWgsuStqgFpFjnPBV
R0LqD6YekvS9oXG+4GrNZnQ0wJYF0dbos+E7lRSdniDf/Ul9rF4WAzAEoQYau8pe
eIPlJy8rVrDEgqfCQabYXrLaG68EHHMGadY2EX0yyI/SZh9AU8RdatNHBwj42LGP
9dp3fyEv14usJPGuLVy+8I7TMckQPpPB+NLFECJMwRRfciPjibw1MMSYTOWnAgMB
AAGjggvZMIIL1TAOBgNVHQ8BAf8EBAMCBaAwgYoGCCsGAQUFBwEBBH4wfDBCBggr
BgEFBQcwAoY2aHR0cDovL3NlY3VyZS5nbG9iYWxzaWduLmNvbS9jYWNlcnQvY2xv
dWRzc2xzaGEyZzMuY3J0MDYGCCsGAQUFBzABhipodHRwOi8vb2NzcDIuZ2xvYmFs
c2lnbi5jb20vY2xvdWRzc2xzaGEyZzMwVgYDVR0gBE8wTTBBBgkrBgEEAaAyARQw
NDAyBggrBgEFBQcCARYmaHR0cHM6Ly93d3cuZ2xvYmFsc2lnbi5jb20vcmVwb3Np
dG9yeS8wCAYGZ4EMAQICMAkGA1UdEwQCMAAwgglrBgNVHREEggliMIIJXoIbcjIu
c2hhcmVkLmdsb2JhbC5mYXN0bHkubmV0ghEqLmFtcGlmeW11c2ljLmNvbYIPKi5h
cGkuZ2lwaHkuY29tghAqLmFwcC5yb213b2QuY29tghAqLmF3YXl0cmF2ZWwuY29t
ghIqLmJmaWZsYXJlbGl2ZS5jb22CECouYmlsbGluZ2FybS5jb22CCiouYnJhemUu
ZXWCFSouY2FsZ2FyeXN0YW1wZWRlLmNvbYIQKi5jZG4udHJpbGxlci5jb4INKi5j
aXR5bWFwcy5pb4IOKi5kZWFsZXJvbi5jb22CDSouZG92ZW1lZC5jb22CDCouZWxh
c3RpYy5jb4IPKi5mZmhhbmRiYWxsLmZyghEqLmZsZXhzaG9wcGVyLmNvbYIPKi5m
bGlwcC1hZHMuY29tghcqLmZsb3JpZGFldmVyYmxhZGVzLmNvbYIYKi5mb2N1c3Jp
dGUtbm92YXRpb24uY29tghAqLmZyZXNoYm9va3MuY29tggsqLmdpcGh5LmNvbYIV
Ki5pZGFob3N0ZWVsaGVhZHMuY29tghAqLmludGVyYWN0bm93LnR2ghEqLmtjbWF2
ZXJpY2tzLmNvbYIMKi5rb21ldHMuY29tghEqLm1lZGlhLmdpcGh5LmNvbYIKKi5t
bnRkLm5ldIIMKi5uYXNjYXIuY29tghUqLm9tbmlnb25wcm9zdWl0ZS5jb22CHSou
b3JsYW5kb3NvbGFyYmVhcnNob2NrZXkuY29tggwqLnByZWlzMjQuZGWCDSoucWEu
bW50ZC5uZXSCEyoucmV2ZXJiLWFzc2V0cy5jb22CDCoucmV2ZXJiLmNvbYIMKi5y
b213b2QuY29tghMqLnNjb290ZXJsb3VuZ2UuY29tghgqLnN0YWdpbmcuYmlsbGlu
Z2Vudi5jb22CFiouc3RhZ2luZy5mcmVzaGVudi5jb22CEiouc3dhbXByYWJiaXRz
LmNvbYILKi52ZXJzZS5jb22CDSoudmlkeWFyZC5jb22CDioudmlld2VkaXQuY29t
ghEqLnZvdGVub3cubmJjLmNvbYIMKi52b3Rlbm93LnR2ggsqLndheWluLmNvbYIb
Ki53ZXN0bWluc3Rlcmtlbm5lbGNsdWIub3Jngg9hbXBpZnltdXNpYy5jb22CE2Fw
aS5yZXZlcmJzaXRlcy5jb22CGGFwaS5zdGFnaW5nLmZyZXNoZW52LmNvbYIbYXBp
LnN0YWdpbmcucmV2ZXJic2l0ZXMuY29tgg5hd2F5dHJhdmVsLmNvbYIQYmZpZmxh
cmVsaXZlLmNvbYITYmZsLXRlc3QuYWJjLmdvLmNvbYIOYmZsLmFiYy5nby5jb22C
CGJyYXplLmV1gh5jZG4taW1hZ2VzLmZsaXBwZW50ZXJwcmlzZS5uZXSCF2Nkbi5m
bGlwcGVudGVycHJpc2UubmV0ghJjb3Ntb3NtYWdhemluZS5jb22CDGRlYWxlcm9u
LmNvbYILZG92ZW1lZC5jb22CHWR3dHN2b3RlLWxpdmUtdGVzdC5hYmMuZ28uY29t
ghhkd3Rzdm90ZS1saXZlLmFiYy5nby5jb22CGGR3dHN2b3RlLXRlc3QuYWJjLmdv
LmNvbYITZHd0c3ZvdGUuYWJjLmdvLmNvbYIKZWxhc3RpYy5jb4IMZW1haWwua2du
LmlvghJmLmNsb3VkLmdpdGh1Yi5jb22CHWZhbmJvb3N0LXRlc3QuZmlhZm9ybXVs
YWUuY29tghhmYW5ib29zdC5maWFmb3JtdWxhZS5jb22CDWZmaGFuZGJhbGwuZnKC
D2ZsZXhzaG9wcGVyLmNvbYIVZmxvcmlkYWV2ZXJibGFkZXMuY29tgglnaXBoeS5j
b22CFWdvLmNvbmNhY2FmbGVhZ3VlLmNvbYIcZ28uY29uY2FjYWZuYXRpb25zbGVh
Z3VlLmNvbYIGZ3BoLmlzghNpZGFob3N0ZWVsaGVhZHMuY29tghNpZG9sdm90ZS5h
YmMuZ28uY29tgg1pbmZyb250LnNwb3J0gg5pbnRlcmFjdG5vdy50doIPa2NtYXZl
cmlja3MuY29tggprb21ldHMuY29tghptYWlsLmRldmVsb3BtZW50LmJyYXplLmNv
bYIWbWFuY2hlc3Rlcm1vbmFyY2hzLmNvbYIWbWVkaWEud29ya2FuZG1vbmV5LmNv
bYIXbXkuc3RhZ2luZy5mcmVzaGVudi5jb22CG29ybGFuZG9zb2xhcmJlYXJzaG9j
a2V5LmNvbYIUcGNhLXRlc3QuZW9ubGluZS5jb22CD3BjYS5lb25saW5lLmNvbYIh
cGxmcGwtZmFzdGx5LnN0YWdpbmcuaXNtZ2FtZXMuY29tggpwcmVpczI0LmRlghRw
cmVtaWVyZXNwZWFrZXJzLmNvbYILcWEudGVub3IuY2+CDHFhLnRlbm9yLmNvbYIe
cm9ib3RpYy1jb29rLnNlY3JldGNkbi1zdGcubmV0ghFzY29vdGVybG91bmdlLmNv
bYIac3RhZ2luZy13d3cuZWFzYS5ldXJvcGEuZXWCGHN0YWdpbmcuZGFpbHkuc3F1
aXJ0Lm9yZ4IUc3RhZ2luZy5mcmVzaGVudi5jb22CEHN3YW1wcmFiYml0cy5jb22C
CHRlbm9yLmNvggl0ZW5vci5jb22CFnRyYWNrLnN3ZWVuZXktbWFpbC5jb22CEHVh
dC5mcmVzaGVudi5jb22CE3VuaWZvcm1zaW5zdG9jay5jb22CF3VzZXJzLnByZW1p
ZXJsZWFndWUuY29tghF1dGFoZ3JpenpsaWVzLmNvbYIJdmVyc2UuY29tggt2aWR5
YXJkLmNvbYIMdmlld2VkaXQuY29tggp2b3Rlbm93LnR2ggl3YXlpbi5jb22CGXdl
c3RtaW5zdGVya2VubmVsY2x1Yi5vcmeCEXd3dy5jaGlxdWVsbGUuY29tghB3d3cu
Y2hpcXVlbGxlLnNlghJ3d3cuZWFzYS5ldXJvcGEuZXWCGnd3dy5pc3JhZWxuYXRp
b25hbG5ld3MuY29tghh3d3cua29nYW5pbnRlcm5ldC5jb20uYXWCDHd3dy50ZW5v
ci5jb4INd3d3LnRlbm9yLmNvbYIUd3d3LnVhdC5mcmVzaGVudi5jb22CF3d3dy51
bmlmb3Jtc2luc3RvY2suY29tghV3d3cudXRhaGdyaXp6bGllcy5jb20wHQYDVR0l
BBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMB8GA1UdIwQYMBaAFKkrh+HOJEc7G7/P
hTcCVZ0NlFjmMB0GA1UdDgQWBBQ7SJi8MbyN4XPx+T1QVj4sLHjhDjCCAQMGCisG
AQQB1nkCBAIEgfQEgfEA7wB1AId1v+dZfPiMQ5lfvfNu/1aNR1Y2/0q1YMG06v9e
oIMPAAABbJgVTmEAAAQDAEYwRAIgeYcRKQDCMIBnswrwBvmmSpCFWhjGl+zabCpo
E3R9nJcCIBaAx/TYKESO7iz+hU6bq7Dwzo0QpTIvho4ZdFfSAAHMAHYAsh4FzIui
zYogTodm+Su5iiUgZ2va+nDnsklTLe+LkF4AAAFsmBVLIQAABAMARzBFAiAfLUaq
ukt75a1pySCxrreQ/+/IAdyOSqXqbH1tZNKlTAIhALlcthwbBCfSSNEjTeJWOXss
clzGt9zAk256uboF0iFLMA0GCSqGSIb3DQEBCwUAA4IBAQCZXc5cmMCeqIVsRnRH
KsuGlT6tP2NdsK1+b9dJguP0zbQoxLg5qBMjRGjDo8BpGOni5mJmRJYDQ/GHKP/d
bd+n/4BDD5jI5/rtl43D+Y1G3S5tCRX/3s+At1LJcuaVRmvnywfE9OLXpI84SWtU
AainsxdCYcvopTOZG9UwkjyuEBV3tVsiQkhRSAzYStM75caRWer2pP7i3AwKNv29
DDSHahXxUyjgAbD2XQojODT/AltEvuqcSrB2cRGXultLmJXFNDEQ5Om4GcjAk75D
pzNLvZuaXHwWoYdm+YTwdPwuZhWe9TxMYlpZbQR8dux2QXRfARF07Vi0+gOzPE9V
RG7L
-----END CERTIFICATE-----`

var elasticChainCert = `-----BEGIN CERTIFICATE-----
MIIEizCCA3OgAwIBAgIORvCM288sVGbvMwHdXzQwDQYJKoZIhvcNAQELBQAwVzEL
MAkGA1UEBhMCQkUxGTAXBgNVBAoTEEdsb2JhbFNpZ24gbnYtc2ExEDAOBgNVBAsT
B1Jvb3QgQ0ExGzAZBgNVBAMTEkdsb2JhbFNpZ24gUm9vdCBDQTAeFw0xNTA4MTkw
MDAwMDBaFw0yNTA4MTkwMDAwMDBaMFcxCzAJBgNVBAYTAkJFMRkwFwYDVQQKExBH
bG9iYWxTaWduIG52LXNhMS0wKwYDVQQDEyRHbG9iYWxTaWduIENsb3VkU1NMIENB
IC0gU0hBMjU2IC0gRzMwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCj
wHXhMpjl2a6EfI3oI19GlVtMoiVw15AEhYDJtfSKZU2Sy6XEQqC2eSUx7fGFIM0T
UT1nrJdNaJszhlyzey2q33egYdH1PPua/NPVlMrJHoAbkJDIrI32YBecMbjFYaLi
blclCG8kmZnPlL/Hi2uwH8oU+hibbBB8mSvaSmPlsk7C/T4QC0j0dwsv8JZLOu69
Nd6FjdoTDs4BxHHT03fFCKZgOSWnJ2lcg9FvdnjuxURbRb0pO+LGCQ+ivivc41za
Wm+O58kHa36hwFOVgongeFxyqGy+Z2ur5zPZh/L4XCf09io7h+/awkfav6zrJ2R7
TFPrNOEvmyBNVBJrfSi9AgMBAAGjggFTMIIBTzAOBgNVHQ8BAf8EBAMCAQYwHQYD
VR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMBIGA1UdEwEB/wQIMAYBAf8CAQAw
HQYDVR0OBBYEFKkrh+HOJEc7G7/PhTcCVZ0NlFjmMB8GA1UdIwQYMBaAFGB7ZhpF
DZfKiVAvfQTNNKj//P1LMD0GCCsGAQUFBwEBBDEwLzAtBggrBgEFBQcwAYYhaHR0
cDovL29jc3AuZ2xvYmFsc2lnbi5jb20vcm9vdHIxMDMGA1UdHwQsMCowKKAmoCSG
Imh0dHA6Ly9jcmwuZ2xvYmFsc2lnbi5jb20vcm9vdC5jcmwwVgYDVR0gBE8wTTAL
BgkrBgEEAaAyARQwPgYGZ4EMAQICMDQwMgYIKwYBBQUHAgEWJmh0dHBzOi8vd3d3
Lmdsb2JhbHNpZ24uY29tL3JlcG9zaXRvcnkvMA0GCSqGSIb3DQEBCwUAA4IBAQCi
HWmKCo7EFIMqKhJNOSeQTvCNrNKWYkc2XpLR+sWTtTcHZSnS9FNQa8n0/jT13bgd
+vzcFKxWlCecQqoETbftWNmZ0knmIC/Tp3e4Koka76fPhi3WU+kLk5xOq9lF7qSE
hf805A7Au6XOX5WJhXCqwV3szyvT2YPfA8qBpwIyt3dhECVO2XTz2XmCtSZwtFK8
jzPXiq4Z0PySrS+6PKBIWEde/SBWlSDBch2rZpmk1Xg3SBufskw3Z3r9QtLTVp7T
HY7EDGiWtkdREPd76xUJZPX58GMWLT3fI0I6k2PMq69PVwbH/hRVYs4nERnh9ELt
IjBrNRpKBYCkZd/My2/Q
-----END CERTIFICATE-----`

func letsEncryptCerts(t *testing.T) (chain []*x509.Certificate) {
	certStrs := []string{
		`-----BEGIN CERTIFICATE-----
MIIEqDCCA5CgAwIBAgISA35dhx5EhUFUA6M+5Ke/a0zOMA0GCSqGSIb3DQEBCwUA
MDIxCzAJBgNVBAYTAlVTMRYwFAYDVQQKEw1MZXQncyBFbmNyeXB0MQswCQYDVQQD
EwJSMzAeFw0yMjEwMDUwMTQwMjRaFw0yMzAxMDMwMTQwMjNaMBQxEjAQBgNVBAMT
CWxlbmNyLm9yZzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABGoPGv9WEV9+30h4
IIjmNVhoSCnRpecjIh05It4+wMFw3DTgEZlWB9u6cR4q/S7vMmU9zTbRx8MLfCnd
oJZjfU+jggKfMIICmzAOBgNVHQ8BAf8EBAMCB4AwHQYDVR0lBBYwFAYIKwYBBQUH
AwEGCCsGAQUFBwMCMAwGA1UdEwEB/wQCMAAwHQYDVR0OBBYEFPY8bUtJRRgcgZ+M
MerILNagQ41PMB8GA1UdIwQYMBaAFBQusxe3WFbLrlAJQOYfr52LFMLGMFUGCCsG
AQUFBwEBBEkwRzAhBggrBgEFBQcwAYYVaHR0cDovL3IzLm8ubGVuY3Iub3JnMCIG
CCsGAQUFBzAChhZodHRwOi8vcjMuaS5sZW5jci5vcmcvMG8GA1UdEQRoMGaCCWxl
bmNyLm9yZ4IPbGV0c2VuY3J5cHQuY29tgg9sZXRzZW5jcnlwdC5vcmeCDXd3dy5s
ZW5jci5vcmeCE3d3dy5sZXRzZW5jcnlwdC5jb22CE3d3dy5sZXRzZW5jcnlwdC5v
cmcwTAYDVR0gBEUwQzAIBgZngQwBAgEwNwYLKwYBBAGC3xMBAQEwKDAmBggrBgEF
BQcCARYaaHR0cDovL2Nwcy5sZXRzZW5jcnlwdC5vcmcwggEEBgorBgEEAdZ5AgQC
BIH1BIHyAPAAdwDoPtDaPvUGNTLnVyi8iWvJA9PL0RFr7Otp4Xd9bQa9bgAAAYOm
BANbAAAEAwBIMEYCIQCChPanUa3D8IkGBs59vfeg3hdrl75pbBU5SZmpLfBIlQIh
ANYzCYs7l+IJxEyVY4uuOG1h33oceoAmBiBxVi+maRLqAHUA36Veq2iCTx9sre64
X04+WurNohKkal6OOxLAIERcKnMAAAGDpgQFVgAABAMARjBEAiALJZvcA8+1kZ44
J/2TYur6bJSVb/EpYK1rHWaSQtRufwIgaiiX+wzGcGLKDS7xMZuQEAeGz0pz/JX9
uEt18al2uD8wDQYJKoZIhvcNAQELBQADggEBAG8Iu74O4iWyejez1Jr3Q18zLv+c
ol5OYSzD1E+ho83SgTkh6js9zkME9Gfjfp7Jp3uJjNMz9BeiKW8phUvPKe6CpLbI
GySH/4rtWW24JDfnxpYw3IIoaYmXxDit7elAcxLGb7pCOE/iuVsikvQOtTmqe9E6
fgScuvPXCBVDQelZNhXBNnT72VTyLOzih1Yx0lnh/58+B86i1XivHAoN1+0pP89D
IqGqv2GhIwyymRhBuqBcVUkXi3gA3glnTcJA2u1i75kMi23veM13BoRltuBXC3gy
u45klaTnRTKwl6ITgTE1buO8dND3wAJ2SCPbXTZnMHtsjKf02bWSp9uc0Hw=
-----END CERTIFICATE-----`,
		`-----BEGIN CERTIFICATE-----
MIIFFjCCAv6gAwIBAgIRAJErCErPDBinU/bWLiWnX1owDQYJKoZIhvcNAQELBQAw
TzELMAkGA1UEBhMCVVMxKTAnBgNVBAoTIEludGVybmV0IFNlY3VyaXR5IFJlc2Vh
cmNoIEdyb3VwMRUwEwYDVQQDEwxJU1JHIFJvb3QgWDEwHhcNMjAwOTA0MDAwMDAw
WhcNMjUwOTE1MTYwMDAwWjAyMQswCQYDVQQGEwJVUzEWMBQGA1UEChMNTGV0J3Mg
RW5jcnlwdDELMAkGA1UEAxMCUjMwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEK
AoIBAQC7AhUozPaglNMPEuyNVZLD+ILxmaZ6QoinXSaqtSu5xUyxr45r+XXIo9cP
R5QUVTVXjJ6oojkZ9YI8QqlObvU7wy7bjcCwXPNZOOftz2nwWgsbvsCUJCWH+jdx
sxPnHKzhm+/b5DtFUkWWqcFTzjTIUu61ru2P3mBw4qVUq7ZtDpelQDRrK9O8Zutm
NHz6a4uPVymZ+DAXXbpyb/uBxa3Shlg9F8fnCbvxK/eG3MHacV3URuPMrSXBiLxg
Z3Vms/EY96Jc5lP/Ooi2R6X/ExjqmAl3P51T+c8B5fWmcBcUr2Ok/5mzk53cU6cG
/kiFHaFpriV1uxPMUgP17VGhi9sVAgMBAAGjggEIMIIBBDAOBgNVHQ8BAf8EBAMC
AYYwHQYDVR0lBBYwFAYIKwYBBQUHAwIGCCsGAQUFBwMBMBIGA1UdEwEB/wQIMAYB
Af8CAQAwHQYDVR0OBBYEFBQusxe3WFbLrlAJQOYfr52LFMLGMB8GA1UdIwQYMBaA
FHm0WeZ7tuXkAXOACIjIGlj26ZtuMDIGCCsGAQUFBwEBBCYwJDAiBggrBgEFBQcw
AoYWaHR0cDovL3gxLmkubGVuY3Iub3JnLzAnBgNVHR8EIDAeMBygGqAYhhZodHRw
Oi8veDEuYy5sZW5jci5vcmcvMCIGA1UdIAQbMBkwCAYGZ4EMAQIBMA0GCysGAQQB
gt8TAQEBMA0GCSqGSIb3DQEBCwUAA4ICAQCFyk5HPqP3hUSFvNVneLKYY611TR6W
PTNlclQtgaDqw+34IL9fzLdwALduO/ZelN7kIJ+m74uyA+eitRY8kc607TkC53wl
ikfmZW4/RvTZ8M6UK+5UzhK8jCdLuMGYL6KvzXGRSgi3yLgjewQtCPkIVz6D2QQz
CkcheAmCJ8MqyJu5zlzyZMjAvnnAT45tRAxekrsu94sQ4egdRCnbWSDtY7kh+BIm
lJNXoB1lBMEKIq4QDUOXoRgffuDghje1WrG9ML+Hbisq/yFOGwXD9RiX8F6sw6W4
avAuvDszue5L3sz85K+EC4Y/wFVDNvZo4TYXao6Z0f+lQKc0t8DQYzk1OXVu8rp2
yJMC6alLbBfODALZvYH7n7do1AZls4I9d1P4jnkDrQoxB3UqQ9hVl3LEKQ73xF1O
yK5GhDDX8oVfGKF5u+decIsH4YaTw7mP3GFxJSqv3+0lUFJoi5Lc5da149p90Ids
hCExroL1+7mryIkXPeFM5TgO9r0rvZaBFOvV2z0gp35Z0+L4WPlbuEjN/lxPFin+
HlUjr8gRsI3qfJOQFy/9rKIJR0Y/8Omwt/8oTWgy1mdeHmmjk7j1nYsvC9JSQ6Zv
MldlTTKB3zhThV1+XWYp6rjd5JW1zbVWEkLNxE7GJThEUG3szgBVGP7pSWTUTsqX
nLRbwHOoq7hHwg==
-----END CERTIFICATE-----`,
		`-----BEGIN CERTIFICATE-----
MIIFYDCCBEigAwIBAgIQQAF3ITfU6UK47naqPGQKtzANBgkqhkiG9w0BAQsFADA/
MSQwIgYDVQQKExtEaWdpdGFsIFNpZ25hdHVyZSBUcnVzdCBDby4xFzAVBgNVBAMT
DkRTVCBSb290IENBIFgzMB4XDTIxMDEyMDE5MTQwM1oXDTI0MDkzMDE4MTQwM1ow
TzELMAkGA1UEBhMCVVMxKTAnBgNVBAoTIEludGVybmV0IFNlY3VyaXR5IFJlc2Vh
cmNoIEdyb3VwMRUwEwYDVQQDEwxJU1JHIFJvb3QgWDEwggIiMA0GCSqGSIb3DQEB
AQUAA4ICDwAwggIKAoICAQCt6CRz9BQ385ueK1coHIe+3LffOJCMbjzmV6B493XC
ov71am72AE8o295ohmxEk7axY/0UEmu/H9LqMZshftEzPLpI9d1537O4/xLxIZpL
wYqGcWlKZmZsj348cL+tKSIG8+TA5oCu4kuPt5l+lAOf00eXfJlII1PoOK5PCm+D
LtFJV4yAdLbaL9A4jXsDcCEbdfIwPPqPrt3aY6vrFk/CjhFLfs8L6P+1dy70sntK
4EwSJQxwjQMpoOFTJOwT2e4ZvxCzSow/iaNhUd6shweU9GNx7C7ib1uYgeGJXDR5
bHbvO5BieebbpJovJsXQEOEO3tkQjhb7t/eo98flAgeYjzYIlefiN5YNNnWe+w5y
sR2bvAP5SQXYgd0FtCrWQemsAXaVCg/Y39W9Eh81LygXbNKYwagJZHduRze6zqxZ
Xmidf3LWicUGQSk+WT7dJvUkyRGnWqNMQB9GoZm1pzpRboY7nn1ypxIFeFntPlF4
FQsDj43QLwWyPntKHEtzBRL8xurgUBN8Q5N0s8p0544fAQjQMNRbcTa0B7rBMDBc
SLeCO5imfWCKoqMpgsy6vYMEG6KDA0Gh1gXxG8K28Kh8hjtGqEgqiNx2mna/H2ql
PRmP6zjzZN7IKw0KKP/32+IVQtQi0Cdd4Xn+GOdwiK1O5tmLOsbdJ1Fu/7xk9TND
TwIDAQABo4IBRjCCAUIwDwYDVR0TAQH/BAUwAwEB/zAOBgNVHQ8BAf8EBAMCAQYw
SwYIKwYBBQUHAQEEPzA9MDsGCCsGAQUFBzAChi9odHRwOi8vYXBwcy5pZGVudHJ1
c3QuY29tL3Jvb3RzL2RzdHJvb3RjYXgzLnA3YzAfBgNVHSMEGDAWgBTEp7Gkeyxx
+tvhS5B1/8QVYIWJEDBUBgNVHSAETTBLMAgGBmeBDAECATA/BgsrBgEEAYLfEwEB
ATAwMC4GCCsGAQUFBwIBFiJodHRwOi8vY3BzLnJvb3QteDEubGV0c2VuY3J5cHQu
b3JnMDwGA1UdHwQ1MDMwMaAvoC2GK2h0dHA6Ly9jcmwuaWRlbnRydXN0LmNvbS9E
U1RST09UQ0FYM0NSTC5jcmwwHQYDVR0OBBYEFHm0WeZ7tuXkAXOACIjIGlj26Ztu
MA0GCSqGSIb3DQEBCwUAA4IBAQAKcwBslm7/DlLQrt2M51oGrS+o44+/yQoDFVDC
5WxCu2+b9LRPwkSICHXM6webFGJueN7sJ7o5XPWioW5WlHAQU7G75K/QosMrAdSW
9MUgNTP52GE24HGNtLi1qoJFlcDyqSMo59ahy2cI2qBDLKobkx/J3vWraV0T9VuG
WCLKTVXkcGdtwlfFRjlBz4pYg1htmf5X6DYO8A4jqv2Il9DjXA6USbW1FzXSLr9O
he8Y4IWS6wY7bCkjCWDcRQJMEhg76fsO3txE+FiYruq9RUWhiF1myv4Q6W+CyBFC
Dfvp7OOGAN6dEOM4+qR9sdjoSYKEBpsr6GtPAQw4dy753ec5
-----END CERTIFICATE-----`,
	}

	for _, s := range certStrs {
		chain = append(chain, parseCert(t, s))
	}
	return chain
}

func letsEncryptXSignedDSTX3(t *testing.T) (certs []*x509.Certificate) {
	certs = letsEncryptCerts(t)
	certs = append(certs, parseCert(t, `-----BEGIN CERTIFICATE-----
MIIDSjCCAjKgAwIBAgIQRK+wgNajJ7qJMDmGLvhAazANBgkqhkiG9w0BAQUFADA/
MSQwIgYDVQQKExtEaWdpdGFsIFNpZ25hdHVyZSBUcnVzdCBDby4xFzAVBgNVBAMT
DkRTVCBSb290IENBIFgzMB4XDTAwMDkzMDIxMTIxOVoXDTIxMDkzMDE0MDExNVow
PzEkMCIGA1UEChMbRGlnaXRhbCBTaWduYXR1cmUgVHJ1c3QgQ28uMRcwFQYDVQQD
Ew5EU1QgUm9vdCBDQSBYMzCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB
AN+v6ZdQCINXtMxiZfaQguzH0yxrMMpb7NnDfcdAwRgUi+DoM3ZJKuM/IUmTrE4O
rz5Iy2Xu/NMhD2XSKtkyj4zl93ewEnu1lcCJo6m67XMuegwGMoOifooUMM0RoOEq
OLl5CjH9UL2AZd+3UWODyOKIYepLYYHsUmu5ouJLGiifSKOeDNoJjj4XLh7dIN9b
xiqKqy69cK3FCxolkHRyxXtqqzTWMIn/5WgTe1QLyNau7Fqckh49ZLOMxt+/yUFw
7BZy1SbsOFU5Q9D8/RhcQPGX69Wam40dutolucbY38EVAjqr2m7xPi71XAicPNaD
aeQQmxkqtilX4+U9m5/wAl0CAwEAAaNCMEAwDwYDVR0TAQH/BAUwAwEB/zAOBgNV
HQ8BAf8EBAMCAQYwHQYDVR0OBBYEFMSnsaR7LHH62+FLkHX/xBVghYkQMA0GCSqG
SIb3DQEBBQUAA4IBAQCjGiybFwBcqR7uKGY3Or+Dxz9LwwmglSBd49lZRNI+DT69
ikugdB/OEIKcdBodfpga3csTS7MgROSR6cz8faXbauX+5v3gTt23ADq1cEmv8uXr
AvHRAosZy5Q6XkjEGB5YGV8eAlrwDPGxrancWYaLbumR9YbK+rlmM6pZW87ipxZz
R8srzJmwN0jP41ZL9c8PDHIyh8bwRLtTcm1D9SZImlJnt1ir/md2cXjbDaJWFBM5
JDGFoqgCWjBH4d1QB7wCCZAA62RjYJsWvIjJEubSfZGL+T0yjWW06XyxV3bqxbYo
Ob8VZRzI9neWagqNdwvYkQsEjgfbKbYK7p2CNTUQ
-----END CERTIFICATE-----`))
	return certs
}
