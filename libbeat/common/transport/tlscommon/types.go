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
	ErrKeyUnspecified = errors.New("key file not configured")

	// ErrKeyNoCertificate indicate a configuration error with missing certificate file
	ErrCertificateUnspecified = errors.New("certificate file not configured")
)

var tlsCipherSuites = map[string]CipherSuite{
	// ECDHE-ECDSA
	"ECDHE-ECDSA-AES-128-CBC-SHA":    CipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA),
	"ECDHE-ECDSA-AES-128-CBC-SHA256": CipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256),
	"ECDHE-ECDSA-AES-128-GCM-SHA256": CipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256),
	"ECDHE-ECDSA-AES-256-CBC-SHA":    CipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA),
	"ECDHE-ECDSA-AES-256-GCM-SHA384": CipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384),
	"ECDHE-ECDSA-CHACHA20-POLY1305":  CipherSuite(tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305),
	"ECDHE-ECDSA-RC4-128-SHA":        CipherSuite(tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA),

	// ECDHE-RSA
	"ECDHE-RSA-3DES-CBC3-SHA":      CipherSuite(tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA),
	"ECDHE-RSA-AES-128-CBC-SHA":    CipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA),
	"ECDHE-RSA-AES-128-CBC-SHA256": CipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256),
	"ECDHE-RSA-AES-128-GCM-SHA256": CipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256),
	"ECDHE-RSA-AES-256-CBC-SHA":    CipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA),
	"ECDHE-RSA-AES-256-GCM-SHA384": CipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384),
	"ECDHE-RSA-CHACHA20-POLY1205":  CipherSuite(tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305),
	"ECDHE-RSA-RC4-128-SHA":        CipherSuite(tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA),

	// RSA-X
	"RSA-RC4-128-SHA":   CipherSuite(tls.TLS_RSA_WITH_RC4_128_SHA),
	"RSA-3DES-CBC3-SHA": CipherSuite(tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA),

	// RSA-AES
	"RSA-AES-128-CBC-SHA":    CipherSuite(tls.TLS_RSA_WITH_AES_128_CBC_SHA),
	"RSA-AES-128-CBC-SHA256": CipherSuite(tls.TLS_RSA_WITH_AES_128_CBC_SHA256),
	"RSA-AES-128-GCM-SHA256": CipherSuite(tls.TLS_RSA_WITH_AES_128_GCM_SHA256),
	"RSA-AES-256-CBC-SHA":    CipherSuite(tls.TLS_RSA_WITH_AES_256_CBC_SHA),
	"RSA-AES-256-GCM-SHA384": CipherSuite(tls.TLS_RSA_WITH_AES_256_GCM_SHA384),

	"TLS-AES-128-GCM-SHA256":       CipherSuite(tls.TLS_AES_128_GCM_SHA256),
	"TLS-AES-256-GCM-SHA384":       CipherSuite(tls.TLS_AES_256_GCM_SHA384),
	"TLS-CHACHA20-POLY1305-SHA256": CipherSuite(tls.TLS_CHACHA20_POLY1305_SHA256),
}

var tlsCipherSuitesInverse = make(map[CipherSuite]string, len(tlsCipherSuites))
var tlsRenegotiationSupportTypesInverse = make(map[TlsRenegotiationSupport]string, len(tlsRenegotiationSupportTypes))
var tlsVerificationModesInverse = make(map[TLSVerificationMode]string, len(tlsVerificationModes))

const unknownString = "unknown"

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
	"P-256":  tlsCurveType(tls.CurveP256),
	"P-384":  tlsCurveType(tls.CurveP384),
	"P-521":  tlsCurveType(tls.CurveP521),
	"X25519": tlsCurveType(tls.X25519),
}

var tlsRenegotiationSupportTypes = map[string]TlsRenegotiationSupport{
	"never":  TlsRenegotiationSupport(tls.RenegotiateNever),
	"once":   TlsRenegotiationSupport(tls.RenegotiateOnceAsClient),
	"freely": TlsRenegotiationSupport(tls.RenegotiateFreelyAsClient),
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

// TLSVerificationMode represents the type of verification to do on the remote host:
// `none`, `certificate`, and `full` and we default to `full`.
// Internally this option is transformed into the `insecure` field in the `tls.Config` struct.
type TLSVerificationMode uint8

// Constants of the supported verification mode.
const (
	VerifyFull TLSVerificationMode = iota
	VerifyNone
	VerifyCertificate
	VerifyStrict
)

var tlsVerificationModes = map[string]TLSVerificationMode{
	"":            VerifyFull,
	"full":        VerifyFull,
	"strict":      VerifyStrict,
	"none":        VerifyNone,
	"certificate": VerifyCertificate,
}

func (m TLSVerificationMode) String() string {
	if s, ok := tlsVerificationModesInverse[m]; ok {
		return s
	}
	return unknownString
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

func (m *tlsClientAuth) Unpack(s string) error {
	mode, found := tlsClientAuthTypes[s]
	if !found {
		return fmt.Errorf("unknown client authentication mode'%v'", s)
	}

	*m = mode
	return nil
}

type CipherSuite uint16

func (cs *CipherSuite) Unpack(s string) error {
	suite, found := tlsCipherSuites[s]
	if !found {
		return fmt.Errorf("invalid tls cipher suite '%v'", s)
	}

	*cs = suite
	return nil
}

func (cs CipherSuite) String() string {
	if s, found := tlsCipherSuitesInverse[cs]; found {
		return s
	}
	return unknownString
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

type TlsRenegotiationSupport tls.RenegotiationSupport

func (r TlsRenegotiationSupport) String() string {
	if t, found := tlsRenegotiationSupportTypesInverse[r]; found {
		return t
	}
	return "<unknown>"
}

func (r *TlsRenegotiationSupport) Unpack(s string) error {
	t, found := tlsRenegotiationSupportTypes[s]
	if !found {
		return fmt.Errorf("invalid tls renegotiation type '%v'", s)
	}

	*r = t
	return nil
}

func (r TlsRenegotiationSupport) MarshalText() ([]byte, error) {
	if t, found := tlsRenegotiationSupportTypesInverse[r]; found {
		return []byte(t), nil
	}

	return nil, fmt.Errorf("could not marshal '%+v' to text", r)
}

func (r TlsRenegotiationSupport) MarshalYAML() (interface{}, error) {
	if t, found := tlsRenegotiationSupportTypesInverse[r]; found {
		return t, nil
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
		return ErrKeyUnspecified
	case !hasCertificate && hasKey:
		return ErrCertificateUnspecified
	}
	return nil
}

func convCipherSuites(suites []CipherSuite) []uint16 {
	if len(suites) == 0 {
		return nil
	}
	cipherSuites := make([]uint16, len(suites))
	for i, s := range suites {
		cipherSuites[i] = uint16(s)
	}
	return cipherSuites
}
