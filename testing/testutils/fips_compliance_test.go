// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build requirefips

package testutils_test

import (
	"testing"

	"github.com/elastic/elastic-agent-libs/testing/fipsscan"
)

const module = "github.com/elastic/beats/v7"

// skipBinaries are package main programs under x-pack/ that are not shipping
// beat binaries and should be excluded from FIPS scanning.
var skipBinaries = []string{
	module + "/x-pack/libbeat",                                                  // mock binary used to test libbeat itself
	module + "/x-pack/filebeat/input/netflow/decoder/examples",                  // example program, not shipped
	module + "/x-pack/filebeat/processors/decode_cef/cef/cmd/cef2json",          // developer tool, not shipped
	module + "/x-pack/heartbeat/monitors/browser/synthexec/testcmd",             // test helper, not shipped
	module + "/x-pack/metricbeat/scripts/msetlists",                             // code-generation script, not shipped
	module + "/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/generate", // code-generation script, not shipped
}

// azidentityViolation is the shared azidentity → x/crypto/pkcs12 violation
// present in every beat via x-pack/libbeat's Azure/AWS identity federation.
var azidentityViolation = []fipsscan.KnownViolation{
	{
		Imported: "golang.org/x/crypto/pkcs12",
		Reason:   "azidentity uses x/crypto/pkcs12 for certificate-based auth; pulled in by x-pack/libbeat Azure/AWS identity federation",
	},
}

// knownViolations documents accepted non-FIPS imports per binary and component.
// Outer key: binary import path, or "" to match all binaries.
// Inner key: component package path or prefix that pulls in the violation, or "" to match any.
// Any violation not listed here will fail the test — add an entry with a Reason
// explaining why the dependency is unavoidable rather than removing it.
var knownViolations = map[string]map[string][]fipsscan.KnownViolation{
	module + "/x-pack/filebeat": {
		// x-pack/libbeat's identity federation pulls in azidentity → x/crypto/pkcs12.
		module + "/x-pack/libbeat/common/identityfederation": azidentityViolation,
		// Azure Event Hub input: AMQP AAD auth and legacy ADAL use x/crypto/pkcs12.
		module + "/x-pack/filebeat/input/azureeventhub": {
			{Imported: "golang.org/x/crypto/pkcs12", Reason: "Azure AMQP AAD auth and legacy ADAL use x/crypto/pkcs12 for certificate handling"},
		},
		// Entity analytics Active Directory provider: go-ldap bundles NTLM
		// (MD4/MD5/DES) which is not FIPS-approved.
		module + "/x-pack/filebeat/input/entityanalytics": {
			{Imported: "github.com/Azure/go-ntlmssp", Reason: "go-ldap bundles NTLM for Active Directory auth; NTLM relies on MD4/MD5/DES — none FIPS-approved"},
			{Imported: "golang.org/x/crypto/md4", Reason: "go-ldap uses x/crypto/md4 for NTLM; MD4 is not FIPS-approved"},
		},
		// GCS input: Google S2A TLS library uses x/crypto for ChaCha20-Poly1305,
		// HKDF, and handshake parsing — none covered by Go's FIPS 140-3 module.
		module + "/x-pack/filebeat/input/gcs": {
			{Imported: "golang.org/x/crypto/chacha20poly1305", Reason: "Google S2A TLS library implements ChaCha20-Poly1305 via x/crypto"},
			{Imported: "golang.org/x/crypto/cryptobyte", Reason: "Google S2A TLS library uses x/crypto/cryptobyte for handshake parsing"},
			{Imported: "golang.org/x/crypto/hkdf", Reason: "Google S2A TLS library uses x/crypto/hkdf for key derivation"},
		},
	},

	module + "/x-pack/metricbeat": {
		// x-pack/libbeat's identity federation pulls in azidentity → x/crypto/pkcs12.
		module + "/x-pack/libbeat/common/identityfederation": azidentityViolation,
		// MySQL module: go-sql-driver uses filippo.io/edwards25519 for Ed25519
		// client authentication, which is not in FIPS 140-3 scope.
		module + "/metricbeat/module/mysql": {
			{Imported: "filippo.io/edwards25519", Reason: "MySQL driver uses filippo.io/edwards25519 for Ed25519 client authentication; not in FIPS 140-3 scope"},
		},
		// GCP modules: Google S2A TLS library uses x/crypto — same as filebeat GCS.
		module + "/x-pack/metricbeat/module/gcp": {
			{Imported: "golang.org/x/crypto/chacha20poly1305", Reason: "Google S2A TLS library implements ChaCha20-Poly1305 via x/crypto"},
			{Imported: "golang.org/x/crypto/cryptobyte", Reason: "Google S2A TLS library uses x/crypto/cryptobyte for handshake parsing"},
			{Imported: "golang.org/x/crypto/hkdf", Reason: "Google S2A TLS library uses x/crypto/hkdf for key derivation"},
		},
	},

	module + "/x-pack/auditbeat": {
		module + "/x-pack/libbeat/common/identityfederation": azidentityViolation,
	},

	module + "/x-pack/heartbeat": {
		module + "/x-pack/libbeat/common/identityfederation": azidentityViolation,
	},

	module + "/x-pack/packetbeat": {
		module + "/x-pack/libbeat/common/identityfederation": azidentityViolation,
	},

	module + "/x-pack/osquerybeat": {
		module + "/x-pack/libbeat/common/identityfederation": azidentityViolation,
	},

	module + "/x-pack/winlogbeat": {
		module + "/x-pack/libbeat/common/identityfederation": azidentityViolation,
	},
}

func TestFIPSFullyCompliant(t *testing.T) {
	fipsscan.CheckModule(t, []string{module + "/x-pack/..."}, skipBinaries, nil, knownViolations)
}
