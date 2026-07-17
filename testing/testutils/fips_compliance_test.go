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
	module + "/x-pack/libbeat",                                                 // mock binary used to test libbeat itself
	module + "/x-pack/filebeat/input/netflow/decoder/examples",                 // example program, not shipped
	module + "/x-pack/filebeat/processors/decode_cef/cef/cmd/cef2json",         // developer tool, not shipped
	module + "/x-pack/heartbeat/monitors/browser/synthexec/testcmd",            // test helper, not shipped
	module + "/x-pack/metricbeat/scripts/msetlists",                            // code-generation script, not shipped
	module + "/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/generate", // code-generation script, not shipped
}

// knownViolations documents accepted non-FIPS imports per binary. Any violation
// not listed here will fail the test — add an entry with a Reason explaining why
// the dependency is unavoidable rather than removing it.
var knownViolations = map[string][]fipsscan.KnownViolation{
	module + "/x-pack/filebeat": {
		{
			Importer: "github.com/Azure/azure-sdk-for-go/sdk/azidentity",
			Imported: "golang.org/x/crypto/pkcs12",
			Reason:   "azidentity uses x/crypto/pkcs12 for certificate-based auth; pulled in by x-pack/libbeat Azure/AWS identity federation",
		},
		{
			Importer: "github.com/Azure/azure-amqp-common-go/v4/aad",
			Imported: "golang.org/x/crypto/pkcs12",
			Reason:   "Azure Event Hub AMQP AAD auth library uses x/crypto/pkcs12 for certificate handling",
		},
		{
			Importer: "github.com/Azure/go-autorest/autorest/adal",
			Imported: "golang.org/x/crypto/pkcs12",
			Reason:   "Azure ADAL (legacy auth) uses x/crypto/pkcs12 for certificate handling",
		},
		{
			Importer: "github.com/go-ldap/ldap/v3",
			Imported: "github.com/Azure/go-ntlmssp",
			Reason:   "Entity analytics Active Directory provider uses go-ldap which bundles NTLM; NTLM relies on MD4/MD5/DES — none FIPS-approved",
		},
		{
			Importer: "github.com/go-ldap/ldap/v3",
			Imported: "golang.org/x/crypto/md4",
			Reason:   "Entity analytics Active Directory provider uses go-ldap which uses x/crypto/md4 for NTLM; MD4 is not FIPS-approved",
		},
		{
			Importer: "github.com/google/s2a-go/internal/record/internal/aeadcrypter",
			Imported: "golang.org/x/crypto/chacha20poly1305",
			Reason:   "Google S2A TLS library implements ChaCha20-Poly1305 via x/crypto; pulled in by GCS input",
		},
		{
			Importer: "github.com/google/s2a-go/internal/record/internal/halfconn",
			Imported: "golang.org/x/crypto/cryptobyte",
			Reason:   "Google S2A TLS library uses x/crypto/cryptobyte for handshake parsing; pulled in by GCS input",
		},
		{
			Importer: "github.com/google/s2a-go/internal/record/internal/halfconn",
			Imported: "golang.org/x/crypto/hkdf",
			Reason:   "Google S2A TLS library uses x/crypto/hkdf for key derivation; pulled in by GCS input",
		},
	},
	module + "/x-pack/metricbeat": {
		{
			Importer: "github.com/Azure/azure-sdk-for-go/sdk/azidentity",
			Imported: "golang.org/x/crypto/pkcs12",
			Reason:   "azidentity uses x/crypto/pkcs12 for certificate-based auth; pulled in by x-pack/libbeat Azure/AWS identity federation",
		},
		{
			Importer: "github.com/go-sql-driver/mysql",
			Imported: "filippo.io/edwards25519",
			Reason:   "MySQL driver imports filippo.io/edwards25519 for Ed25519 client authentication; not in FIPS 140-3 scope",
		},
		{
			Importer: "github.com/google/s2a-go/internal/record/internal/aeadcrypter",
			Imported: "golang.org/x/crypto/chacha20poly1305",
			Reason:   "Google S2A TLS library implements ChaCha20-Poly1305 via x/crypto; pulled in by GCP modules",
		},
		{
			Importer: "github.com/google/s2a-go/internal/record/internal/halfconn",
			Imported: "golang.org/x/crypto/cryptobyte",
			Reason:   "Google S2A TLS library uses x/crypto/cryptobyte for handshake parsing; pulled in by GCP modules",
		},
		{
			Importer: "github.com/google/s2a-go/internal/record/internal/halfconn",
			Imported: "golang.org/x/crypto/hkdf",
			Reason:   "Google S2A TLS library uses x/crypto/hkdf for key derivation; pulled in by GCP modules",
		},
	},
	module + "/x-pack/auditbeat": {
		{
			Importer: "github.com/Azure/azure-sdk-for-go/sdk/azidentity",
			Imported: "golang.org/x/crypto/pkcs12",
			Reason:   "azidentity uses x/crypto/pkcs12 for certificate-based auth; pulled in by x-pack/libbeat Azure/AWS identity federation",
		},
	},
	module + "/x-pack/heartbeat": {
		{
			Importer: "github.com/Azure/azure-sdk-for-go/sdk/azidentity",
			Imported: "golang.org/x/crypto/pkcs12",
			Reason:   "azidentity uses x/crypto/pkcs12 for certificate-based auth; pulled in by x-pack/libbeat Azure/AWS identity federation",
		},
	},
	module + "/x-pack/packetbeat": {
		{
			Importer: "github.com/Azure/azure-sdk-for-go/sdk/azidentity",
			Imported: "golang.org/x/crypto/pkcs12",
			Reason:   "azidentity uses x/crypto/pkcs12 for certificate-based auth; pulled in by x-pack/libbeat Azure/AWS identity federation",
		},
	},
	module + "/x-pack/osquerybeat": {
		{
			Importer: "github.com/Azure/azure-sdk-for-go/sdk/azidentity",
			Imported: "golang.org/x/crypto/pkcs12",
			Reason:   "azidentity uses x/crypto/pkcs12 for certificate-based auth; pulled in by x-pack/libbeat Azure/AWS identity federation",
		},
	},
	module + "/x-pack/winlogbeat": {
		{
			Importer: "github.com/Azure/azure-sdk-for-go/sdk/azidentity",
			Imported: "golang.org/x/crypto/pkcs12",
			Reason:   "azidentity uses x/crypto/pkcs12 for certificate-based auth; pulled in by x-pack/libbeat Azure/AWS identity federation",
		},
	},
}

func TestFIPSFullyCompliant(t *testing.T) {
	fipsscan.CheckModule(t, []string{module + "/x-pack/..."}, skipBinaries, nil, knownViolations)
}
