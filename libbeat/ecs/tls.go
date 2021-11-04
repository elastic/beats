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

// Fields related to a TLS connection. These fields focus on the TLS protocol
// itself and intentionally avoids in-depth analysis of the related x.509
// certificate files.
type Tls struct {
	// Numeric part of the version parsed from the original string.
	Version string `ecs:"version"`

	// Normalized lowercase protocol name parsed from original string.
	VersionProtocol string `ecs:"version_protocol"`

	// String indicating the cipher used during the current connection.
	Cipher string `ecs:"cipher"`

	// String indicating the curve used for the given cipher, when applicable.
	Curve string `ecs:"curve"`

	// Boolean flag indicating if this TLS connection was resumed from an
	// existing TLS negotiation.
	Resumed bool `ecs:"resumed"`

	// Boolean flag indicating if the TLS negotiation was successful and
	// transitioned to an encrypted tunnel.
	Established bool `ecs:"established"`

	// String indicating the protocol being tunneled. Per the values in the
	// IANA registry
	// (https://www.iana.org/assignments/tls-extensiontype-values/tls-extensiontype-values.xhtml#alpn-protocol-ids),
	// this string should be lower case.
	NextProtocol string `ecs:"next_protocol"`

	// A hash that identifies clients based on how they perform an SSL/TLS
	// handshake.
	ClientJa3 string `ecs:"client.ja3"`

	// Also called an SNI, this tells the server which hostname to which the
	// client is attempting to connect to. When this value is available, it
	// should get copied to `destination.domain`.
	ClientServerName string `ecs:"client.server_name"`

	// Array of ciphers offered by the client during the client hello.
	ClientSupportedCiphers []string `ecs:"client.supported_ciphers"`

	// Distinguished name of subject of the x.509 certificate presented by the
	// client.
	ClientSubject string `ecs:"client.subject"`

	// Distinguished name of subject of the issuer of the x.509 certificate
	// presented by the client.
	ClientIssuer string `ecs:"client.issuer"`

	// Date/Time indicating when client certificate is first considered valid.
	ClientNotBefore time.Time `ecs:"client.not_before"`

	// Date/Time indicating when client certificate is no longer considered
	// valid.
	ClientNotAfter time.Time `ecs:"client.not_after"`

	// Array of PEM-encoded certificates that make up the certificate chain
	// offered by the client. This is usually mutually-exclusive of
	// `client.certificate` since that value should be the first certificate in
	// the chain.
	ClientCertificateChain []string `ecs:"client.certificate_chain"`

	// PEM-encoded stand-alone certificate offered by the client. This is
	// usually mutually-exclusive of `client.certificate_chain` since this
	// value also exists in that list.
	ClientCertificate string `ecs:"client.certificate"`

	// Certificate fingerprint using the MD5 digest of DER-encoded version of
	// certificate offered by the client. For consistency with other hash
	// values, this value should be formatted as an uppercase hash.
	ClientHashMd5 string `ecs:"client.hash.md5"`

	// Certificate fingerprint using the SHA1 digest of DER-encoded version of
	// certificate offered by the client. For consistency with other hash
	// values, this value should be formatted as an uppercase hash.
	ClientHashSha1 string `ecs:"client.hash.sha1"`

	// Certificate fingerprint using the SHA256 digest of DER-encoded version
	// of certificate offered by the client. For consistency with other hash
	// values, this value should be formatted as an uppercase hash.
	ClientHashSha256 string `ecs:"client.hash.sha256"`

	// A hash that identifies servers based on how they perform an SSL/TLS
	// handshake.
	ServerJa3s string `ecs:"server.ja3s"`

	// Subject of the x.509 certificate presented by the server.
	ServerSubject string `ecs:"server.subject"`

	// Subject of the issuer of the x.509 certificate presented by the server.
	ServerIssuer string `ecs:"server.issuer"`

	// Timestamp indicating when server certificate is first considered valid.
	ServerNotBefore time.Time `ecs:"server.not_before"`

	// Timestamp indicating when server certificate is no longer considered
	// valid.
	ServerNotAfter time.Time `ecs:"server.not_after"`

	// Array of PEM-encoded certificates that make up the certificate chain
	// offered by the server. This is usually mutually-exclusive of
	// `server.certificate` since that value should be the first certificate in
	// the chain.
	ServerCertificateChain []string `ecs:"server.certificate_chain"`

	// PEM-encoded stand-alone certificate offered by the server. This is
	// usually mutually-exclusive of `server.certificate_chain` since this
	// value also exists in that list.
	ServerCertificate string `ecs:"server.certificate"`

	// Certificate fingerprint using the MD5 digest of DER-encoded version of
	// certificate offered by the server. For consistency with other hash
	// values, this value should be formatted as an uppercase hash.
	ServerHashMd5 string `ecs:"server.hash.md5"`

	// Certificate fingerprint using the SHA1 digest of DER-encoded version of
	// certificate offered by the server. For consistency with other hash
	// values, this value should be formatted as an uppercase hash.
	ServerHashSha1 string `ecs:"server.hash.sha1"`

	// Certificate fingerprint using the SHA256 digest of DER-encoded version
	// of certificate offered by the server. For consistency with other hash
	// values, this value should be formatted as an uppercase hash.
	ServerHashSha256 string `ecs:"server.hash.sha256"`
}
