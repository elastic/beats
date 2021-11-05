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

package ecs

import (
	"time"
)

// This implements the common core fields for x509 certificates. This
// information is likely logged with TLS sessions, digital signatures found in
// executable binaries, S/MIME information in email bodies, or analysis of
// files on disk.
// When the certificate relates to a file, use the fields at `file.x509`. When
// hashes of the DER-encoded certificate are available, the `hash` data set
// should be populated as well (e.g. `file.hash.sha256`).
// Events that contain certificate information about network connections,
// should use the x509 fields under the relevant TLS fields: `tls.server.x509`
// and/or `tls.client.x509`.
type X509 struct {
	// Version of x509 format.
	VersionNumber string `ecs:"version_number"`

	// Unique serial number issued by the certificate authority. For
	// consistency, if this value is alphanumeric, it should be formatted
	// without colons and uppercase characters.
	SerialNumber string `ecs:"serial_number"`

	// Distinguished name (DN) of issuing certificate authority.
	IssuerDistinguishedName string `ecs:"issuer.distinguished_name"`

	// List of common name (CN) of issuing certificate authority.
	IssuerCommonName string `ecs:"issuer.common_name"`

	// List of organizational units (OU) of issuing certificate authority.
	IssuerOrganizationalUnit string `ecs:"issuer.organizational_unit"`

	// List of organizations (O) of issuing certificate authority.
	IssuerOrganization string `ecs:"issuer.organization"`

	// List of locality names (L)
	IssuerLocality string `ecs:"issuer.locality"`

	// List of state or province names (ST, S, or P)
	IssuerStateOrProvince string `ecs:"issuer.state_or_province"`

	// List of country (C) codes
	IssuerCountry string `ecs:"issuer.country"`

	// Identifier for certificate signature algorithm. We recommend using names
	// found in Go Lang Crypto library. See
	// https://github.com/golang/go/blob/go1.14/src/crypto/x509/x509.go#L337-L353.
	SignatureAlgorithm string `ecs:"signature_algorithm"`

	// Time at which the certificate is first considered valid.
	NotBefore time.Time `ecs:"not_before"`

	// Time at which the certificate is no longer considered valid.
	NotAfter time.Time `ecs:"not_after"`

	// Distinguished name (DN) of the certificate subject entity.
	SubjectDistinguishedName string `ecs:"subject.distinguished_name"`

	// List of common names (CN) of subject.
	SubjectCommonName string `ecs:"subject.common_name"`

	// List of organizational units (OU) of subject.
	SubjectOrganizationalUnit string `ecs:"subject.organizational_unit"`

	// List of organizations (O) of subject.
	SubjectOrganization string `ecs:"subject.organization"`

	// List of locality names (L)
	SubjectLocality string `ecs:"subject.locality"`

	// List of state or province names (ST, S, or P)
	SubjectStateOrProvince string `ecs:"subject.state_or_province"`

	// List of country (C) code
	SubjectCountry string `ecs:"subject.country"`

	// Algorithm used to generate the public key.
	PublicKeyAlgorithm string `ecs:"public_key_algorithm"`

	// The size of the public key space in bits.
	PublicKeySize int64 `ecs:"public_key_size"`

	// Exponent used to derive the public key. This is algorithm specific.
	PublicKeyExponent int64 `ecs:"public_key_exponent"`

	// The curve used by the elliptic curve public key algorithm. This is
	// algorithm specific.
	PublicKeyCurve string `ecs:"public_key_curve"`

	// List of subject alternative names (SAN). Name types vary by certificate
	// authority and certificate type but commonly contain IP addresses, DNS
	// names (and wildcards), and email addresses.
	AlternativeNames string `ecs:"alternative_names"`
}
