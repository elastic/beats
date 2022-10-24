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
	dsa2 "crypto/dsa" //nolint:staticcheck // we need to calculate DSA stuff for completeness
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	cryptoTLS "crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/heartbeat/look"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

// UnknownTLSHandshakeDuration to be used in AddTLSMetadata when the duration of the TLS handshake can't be determined.
const UnknownTLSHandshakeDuration = time.Duration(-1)

func AddTLSMetadata(fields mapstr.M, connState cryptoTLS.ConnectionState, duration time.Duration) {
	_, _ = fields.Put("tls.established", true)
	if duration != UnknownTLSHandshakeDuration {
		_, _ = fields.Put("tls.rtt.handshake", look.RTT(duration))
	}
	versionDetails := tlscommon.TLSVersion(connState.Version).Details()
	// The only situation in which versionDetails would be nil is if an unknown TLS version were to be
	// encountered. Not filling the fields here makes sense, since there's no standard 'unknown' value.
	if versionDetails != nil {
		_, _ = fields.Put("tls.version_protocol", versionDetails.Protocol)
		_, _ = fields.Put("tls.version", versionDetails.Version)
	}

	if connState.NegotiatedProtocol != "" {
		_, _ = fields.Put("tls.next_protocol", connState.NegotiatedProtocol)
	}
	_, _ = fields.Put("tls.cipher", tlscommon.ResolveCipherSuite(connState.CipherSuite))

	tlsFields := CertFields(connState.PeerCertificates[0], connState.VerifiedChains)

	fields.DeepUpdate(mapstr.M{"tls": tlsFields})
}

func CertFields(hostCert *x509.Certificate, verifiedChains [][]*x509.Certificate) (tlsFields mapstr.M) {
	x509Fields := mapstr.M{}
	serverFields := mapstr.M{"x509": x509Fields}
	tlsFields = mapstr.M{"server": serverFields}

	_, _ = serverFields.Put("hash.sha1", fmt.Sprintf("%x", sha1.Sum(hostCert.Raw)))
	_, _ = serverFields.Put("hash.sha256", fmt.Sprintf("%x", sha256.Sum256(hostCert.Raw)))

	_, _ = x509Fields.Put("issuer.common_name", hostCert.Issuer.CommonName)
	_, _ = x509Fields.Put("issuer.distinguished_name", hostCert.Issuer.String())
	_, _ = x509Fields.Put("subject.common_name", hostCert.Subject.CommonName)
	_, _ = x509Fields.Put("subject.distinguished_name", hostCert.Subject.String())
	_, _ = x509Fields.Put("serial_number", hostCert.SerialNumber.String())
	_, _ = x509Fields.Put("signature_algorithm", hostCert.SignatureAlgorithm.String())
	_, _ = x509Fields.Put("public_key_algorithm", hostCert.PublicKeyAlgorithm.String())
	_, _ = x509Fields.Put("not_before", hostCert.NotBefore)
	_, _ = tlsFields.Put("certificate_not_valid_before", hostCert.NotBefore)
	_, _ = x509Fields.Put("not_after", hostCert.NotAfter)
	_, _ = tlsFields.Put("certificate_not_valid_after", hostCert.NotAfter)
	if rsaKey, ok := hostCert.PublicKey.(*rsa.PublicKey); ok {
		sizeInBits := rsaKey.Size() * 8
		_, _ = x509Fields.Put("public_key_size", sizeInBits)
		_, _ = x509Fields.Put("public_key_exponent", rsaKey.E)
	} else if dsaKey, ok := hostCert.PublicKey.(*dsa2.PublicKey); ok {
		if dsaKey.Parameters.P != nil {
			_, _ = x509Fields.Put("public_key_size", len(dsaKey.P.Bytes())*8)
		} else {
			_, _ = x509Fields.Put("public_key_size", len(dsaKey.P.Bytes())*8)
		}
	} else if ecdsa, ok := hostCert.PublicKey.(*ecdsa.PublicKey); ok {
		_, _ = x509Fields.Put("public_key_curve", ecdsa.Curve.Params().Name)
	}

	// If we have fully verified cert chains, use those for the
	// not_before / not_after timestamps
	//
	// we compute the soonest point at which this cert chain will become invalid
	// this only happens when strict verification is enabled
	// due to the implementation in elastic-agent-libs
	// which only gives us the chain metadata in that scenario, unlike
	// the go stdlib
	// https://github.com/elastic/elastic-agent-libs/blob/main/transport/tlscommon/tls_config.go#L240
	var latestChainExpiration time.Time
	now := time.Now()
	for _, chain := range verifiedChains {
		chainNotBefore, chainNotAfter := calculateCertTimestamps(chain)

		// If this chain expires sooner than a previously seen chain we don't
		// set any fields
		if chainNotAfter != nil {
			if chainNotAfter.Before(latestChainExpiration) && chainNotBefore.After(now) {
				continue
			}
			latestChainExpiration = *chainNotAfter
		}

		// Legacy non-ECS field
		_, _ = tlsFields.Put("certificate_not_valid_before", chainNotBefore)
		_, _ = x509Fields.Put("not_before", chainNotBefore)
		if chainNotAfter != nil {
			// Legacy non-ECS field
			_, _ = tlsFields.Put("certificate_not_valid_after", *chainNotAfter)
			_, _ = x509Fields.Put("not_after", *chainNotAfter)
		}
	}

	return tlsFields
}

func calculateCertTimestamps(certs []*x509.Certificate) (chainNotBefore time.Time, chainNotAfter *time.Time) {
	// The behavior here might seem strange. We *always* set a notBefore, but only optionally set a notAfter.
	// Why might we do this?
	// The root cause is that the x509.Certificate type uses time.Time for these tlsFields instead of *time.Time
	// so we have no way to know if the user actually set these tlsFields. The x509 RFC says that only one of the
	// two tlsFields must be set. Most tools (including openssl and go's certgen) always set both. BECAUSE WHY NOT
	//
	// In the wild, however, there are certs missing one of these two tlsFields.
	// So, what's the correct behavior here? We cannot know if a field was omitted due to the lack of nullability.
	// So, in this case, we try to do what people will want 99.99999999999999999% of the time.
	// People might set notBefore to go's zero date intentionally when creating certs. So, we always set that
	// field, even if we find a zero value.
	// However, it would be weird to set notAfter to the zero value. That could invalidate a cert that was intended
	// to be valid forever. So, in that case, we treat the zero value as non-existent.
	// This is why notBefore is a time.Time and notAfter is a *time.Time

	// We need the zero date later
	var zeroTime time.Time

	// Here we compute the minimal bounds during which this certificate chain is valid
	// To do this correctly, we take the maximum NotBefore and the minimum NotAfter.
	// This *should* always wind up being the terminal cert in the chain, but we should
	// compute this correctly.
	for _, cert := range certs {
		if chainNotBefore.Before(cert.NotBefore) {
			chainNotBefore = cert.NotBefore
		}

		if cert.NotAfter != zeroTime && (chainNotAfter == nil || chainNotAfter.After(cert.NotAfter)) {
			chainNotAfter = &cert.NotAfter
		}
	}

	return chainNotBefore, chainNotAfter
}
