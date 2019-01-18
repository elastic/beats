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
	"crypto/tls"
	"errors"
	"fmt"
)

var (
	// ErrNotACertificate indicates a PEM file to be loaded not being a valid
	// PEM file or certificate.
	ErrNotACertificate = errors.New("file is not a certificate")

	// ErrCertificateNoKey indicate a configuration error with missing key file
	ErrCertificateNoKey = errors.New("key file not configured")

	// ErrKeyNoCertificate indicate a configuration error with missing certificate file
	ErrKeyNoCertificate = errors.New("certificate file not configured")
)

var tlsCipherSuites = map[string]tlsCipherSuite{
	"ECDHE-ECDSA-AES-128-CBC-SHA":    tlsCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA),
	"ECDHE-ECDSA-AES-128-GCM-SHA256": tlsCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256),
	"ECDHE-ECDSA-AES-256-CBC-SHA":    tlsCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA),
	"ECDHE-ECDSA-AES-256-GCM-SHA384": tlsCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384),
	"ECDHE-ECDSA-RC4-128-SHA":        tlsCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA),
	"ECDHE-RSA-3DES-CBC3-SHA":        tlsCipherSuite(tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA),
	"ECDHE-RSA-AES-128-CBC-SHA":      tlsCipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA),
	"ECDHE-RSA-AES-128-GCM-SHA256":   tlsCipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256),
	"ECDHE-RSA-AES-256-CBC-SHA":      tlsCipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA),
	"ECDHE-RSA-AES-256-GCM-SHA384":   tlsCipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384),
	"ECDHE-RSA-RC4-128-SHA":          tlsCipherSuite(tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA),
	"RSA-3DES-CBC3-SHA":              tlsCipherSuite(tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA),
	"RSA-AES-128-CBC-SHA":            tlsCipherSuite(tls.TLS_RSA_WITH_AES_128_CBC_SHA),
	"RSA-AES-128-GCM-SHA256":         tlsCipherSuite(tls.TLS_RSA_WITH_AES_128_GCM_SHA256),
	"RSA-AES-256-CBC-SHA":            tlsCipherSuite(tls.TLS_RSA_WITH_AES_256_CBC_SHA),
	"RSA-AES-256-GCM-SHA384":         tlsCipherSuite(tls.TLS_RSA_WITH_AES_256_GCM_SHA384),
	"RSA-RC4-128-SHA":                tlsCipherSuite(tls.TLS_RSA_WITH_RC4_128_SHA),
}

var tlsCipherSuitesInverse = make(map[tlsCipherSuite]string, len(tlsCipherSuites))
var tlsRenegotiationSupportTypesInverse = make(map[tlsRenegotiationSupport]string, len(tlsRenegotiationSupportTypes))
var tlsVerificationModesInverse = make(map[TLSVerificationMode]string, len(tlsVerificationModes))

// Init creates a inverse representation of the values mapping.
func init() {
	for cipherName, i := range tlsCipherSuites {
		tlsCipherSuitesInverse[i] = cipherName
	}

	for name, t := range tlsRenegotiationSupportTypes {
		tlsRenegotiationSupportTypesInverse[t] = name
	}

	for name, t := range tlsVerificationModes {
		tlsVerificationModesInverse[t] = name
	}
}

var tlsCurveTypes = map[string]tlsCurveType{
	"P-256": tlsCurveType(tls.CurveP256),
	"P-384": tlsCurveType(tls.CurveP384),
	"P-521": tlsCurveType(tls.CurveP521),
}

var tlsRenegotiationSupportTypes = map[string]tlsRenegotiationSupport{
	"never":  tlsRenegotiationSupport(tls.RenegotiateNever),
	"once":   tlsRenegotiationSupport(tls.RenegotiateOnceAsClient),
	"freely": tlsRenegotiationSupport(tls.RenegotiateFreelyAsClient),
}

// TLSVersion type for TLS version.
type TLSVersion uint16

// Define all the possible TLS version.
const (
	TLSVersionSSL30 TLSVersion = tls.VersionSSL30
	TLSVersion10    TLSVersion = tls.VersionTLS10
	TLSVersion11    TLSVersion = tls.VersionTLS11
	TLSVersion12    TLSVersion = tls.VersionTLS12
)

// TLSDefaultVersions list of versions of TLS we should support.
var TLSDefaultVersions = []TLSVersion{
	TLSVersion10,
	TLSVersion11,
	TLSVersion12,
}

type tlsClientAuth int

const (
	tlsClientAuthNone     tlsClientAuth = tlsClientAuth(tls.NoClientCert)
	tlsClientAuthOptional               = tlsClientAuth(tls.VerifyClientCertIfGiven)
	tlsClientAuthRequired               = tlsClientAuth(tls.RequireAndVerifyClientCert)
)

var tlsClientAuthTypes = map[string]tlsClientAuth{
	"none":     tlsClientAuthNone,
	"optional": tlsClientAuthOptional,
	"required": tlsClientAuthRequired,
}

var tlsProtocolVersions = map[string]TLSVersion{
	"SSLv3":   TLSVersionSSL30,
	"SSLv3.0": TLSVersionSSL30,
	"TLSv1":   TLSVersion10,
	"TLSv1.0": TLSVersion10,
	"TLSv1.1": TLSVersion11,
	"TLSv1.2": TLSVersion12,
}

var tlsProtocolVersionsInverse = map[TLSVersion]string{
	TLSVersionSSL30: "SSLv3",
	TLSVersion10:    "TLSv1.0",
	TLSVersion11:    "TLSv1.1",
	TLSVersion12:    "TLSv1.2",
}

// TLSVerificationMode represents the type of verification to do on the remote host,
// `none` or `full` and we default to `full`, internally this option is transformed into the
// `insecure` field in the `tls.Config` struct.
type TLSVerificationMode uint8

// Constants of the supported verification mode.
const (
	VerifyFull TLSVerificationMode = iota
	VerifyNone

	// TODO: add VerifyCertificate support. Due to checks being run
	//       during handshake being limited, verify certificates in
	//       postVerifyTLSConnection
	// VerifyCertificate
)

func (v TLSVersion) String() string {
	if s, ok := tlsProtocolVersionsInverse[v]; ok {
		return s
	}
	return "unknown"
}

//Unpack transforms the string into a constant.
func (v *TLSVersion) Unpack(s string) error {
	version, found := tlsProtocolVersions[s]
	if !found {
		return fmt.Errorf("invalid tls version '%v'", s)
	}

	*v = version
	return nil
}

var tlsVerificationModes = map[string]TLSVerificationMode{
	"":     VerifyFull,
	"full": VerifyFull,
	"none": VerifyNone,
	// "certificate": verifyCertificate,
}

func (m TLSVerificationMode) String() string {
	if s, ok := tlsVerificationModesInverse[m]; ok {
		return s
	}
	return "unknown"
}

// MarshalText marshal the verification mode into a human readable value.
func (m TLSVerificationMode) MarshalText() ([]byte, error) {
	if s, ok := tlsVerificationModesInverse[m]; ok {
		return []byte(s), nil
	}
	return nil, fmt.Errorf("could not marshal '%+v' to text", m)
}

// Unpack unpacks the string into constants.
func (m *TLSVerificationMode) Unpack(in interface{}) error {
	if in == nil {
		*m = VerifyFull
		return nil
	}

	s, ok := in.(string)
	if !ok {
		return fmt.Errorf("verification mode must be an identifier")
	}

	mode, found := tlsVerificationModes[s]
	if !found {
		return fmt.Errorf("unknown verification mode '%v'", s)
	}

	*m = mode
	return nil
}

func (m *tlsClientAuth) Unpack(in interface{}) error {
	if in == nil {
		*m = tlsClientAuthRequired
		return nil
	}

	s, ok := in.(string)
	if !ok {
		return fmt.Errorf("client authentication must be an identifier")
	}

	mode, found := tlsClientAuthTypes[s]
	if !found {
		return fmt.Errorf("unknown client authentication mode'%v'", s)
	}

	*m = mode
	return nil
}

type tlsCipherSuite uint16

func (cs *tlsCipherSuite) Unpack(s string) error {
	suite, found := tlsCipherSuites[s]
	if !found {
		return fmt.Errorf("invalid tls cipher suite '%v'", s)
	}

	*cs = suite
	return nil
}

func (cs tlsCipherSuite) String() string {
	if s, found := tlsCipherSuitesInverse[cs]; found {
		return s
	}
	return "unknown"
}

type tlsCurveType tls.CurveID

func (ct *tlsCurveType) Unpack(s string) error {
	t, found := tlsCurveTypes[s]
	if !found {
		return fmt.Errorf("invalid tls curve type '%v'", s)
	}

	*ct = t
	return nil
}

type tlsRenegotiationSupport tls.RenegotiationSupport

func (r *tlsRenegotiationSupport) Unpack(s string) error {
	t, found := tlsRenegotiationSupportTypes[s]
	if !found {
		return fmt.Errorf("invalid tls renegotiation type '%v'", s)
	}

	*r = t
	return nil
}

func (r tlsRenegotiationSupport) MarshalText() ([]byte, error) {
	if t, found := tlsRenegotiationSupportTypesInverse[r]; found {
		return []byte(t), nil
	}

	return nil, fmt.Errorf("could not marshal '%+v' to text", r)
}

// CertificateConfig define a common set of fields for a certificate.
type CertificateConfig struct {
	Certificate string `config:"certificate" yaml:"certificate,omitempty"`
	Key         string `config:"key" yaml:"key,omitempty"`
	Passphrase  string `config:"key_passphrase" yaml:"key_passphrase,omitempty"`
}

// Validate validates the CertificateConfig
func (c *CertificateConfig) Validate() error {
	hasCertificate := c.Certificate != ""
	hasKey := c.Key != ""

	switch {
	case hasCertificate && !hasKey:
		return ErrCertificateNoKey
	case !hasCertificate && hasKey:
		return ErrKeyNoCertificate
	}
	return nil
}
