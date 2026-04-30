// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package elasticsearchstorage

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
)

// isValidESIndexName mirrors ES's documented index-naming rules
// (https://www.elastic.co/docs/api/doc/elasticsearch/operation/operation-indices-create).
// Used to assert the sanitizer produces valid output for any input.
func isValidESIndexName(s string) bool {
	if len(s) == 0 || len(s) > 255 {
		return false
	}
	switch s[0] {
	case '-', '_', '+':
		return false
	}
	if s == "." || s == ".." {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			return false
		case r == '\\', r == '/', r == '*', r == '?', r == '"',
			r == '<', r == '>', r == '|', r == ',', r == '#', r == ':':
			return false
		case r == ' ', r == '\t', r == '\n', r == '\r':
			return false
		}
	}
	return true
}

func TestSanitizeIndexSuffix(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // "" means we don't assert exact value, only ES-validity and non-empty
	}{
		{"plain", "akamai_siem_raw", "akamai_siem_raw"},
		{"slash from named instance", "receiver_akamai_siem/raw", "receiver_akamai_siem-raw"},
		{"uppercase", "Receiver_Akamai_SIEM", "receiver_akamai_siem"},
		{"mixed punctuation", "foo*bar?baz", "foo-bar-baz"},
		{"leading underscore stripped", "_starts_underscore", "starts_underscore"},
		{"leading dash stripped", "---weird", "weird"},
		{"all illegal falls back to hash", `\/*?"<>|,#: `, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeIndexSuffix(tc.in)
			if tc.want != "" {
				assert.Equal(t, tc.want, got)
			}
			assert.NotEmpty(t, got, "sanitizer must never return an empty suffix")
			assert.True(t, isValidESIndexName(indexNamePrefix+got),
				"sanitized index name must be ES-valid: %q", indexNamePrefix+got)
		})
	}
}

func TestSanitizeIndexSuffix_AllIllegal_DistinctInputs_DistinctOutputs(t *testing.T) {
	// Two different degenerate inputs must still land in different
	// indices — the hash fallback provides this guarantee.
	a := sanitizeIndexSuffix(`\/*?"<>`)
	b := sanitizeIndexSuffix(`|,#: `)
	assert.NotEqual(t, a, b)
}

func TestSanitizeIndexSuffix_LongName_Truncates(t *testing.T) {
	// Construct a name longer than 200 bytes; sanitizer should hash-truncate.
	long := strings.Repeat("a", 400)
	got := sanitizeIndexSuffix(long)

	assert.LessOrEqual(t, len(got), 200, "sanitized suffix must fit within the 200-byte budget")
	assert.True(t, isValidESIndexName(indexNamePrefix+got))

	// Two distinct long inputs must produce distinct outputs (the hash
	// portion provides this property).
	other := strings.Repeat("b", 400)
	gotOther := sanitizeIndexSuffix(other)
	assert.NotEqual(t, got, gotOther)
}

func TestComposeIndexName_NamedReceivers(t *testing.T) {
	// The motivating case from elastic/beats#50223: two receivers of the
	// same type but different names must land in distinct, valid indices.
	idRaw := component.MustNewIDWithName("akamai_siem", "raw")
	idOtel := component.MustNewIDWithName("akamai_siem", "otel")

	rawName := composeIndexName(component.KindReceiver, idRaw, "")
	otelName := composeIndexName(component.KindReceiver, idOtel, "")

	assert.Equal(t, "agentless-state-receiver_akamai_siem_raw", rawName)
	assert.Equal(t, "agentless-state-receiver_akamai_siem_otel", otelName)
	assert.True(t, isValidESIndexName(rawName))
	assert.True(t, isValidESIndexName(otelName))
}

func TestComposeIndexName_StorageNameDisambiguates(t *testing.T) {
	// A consumer with multiple per-signal storages (e.g. logs/metrics)
	// passes a non-empty storageName. The composed index name must
	// include it so signals don't collide.
	id := component.MustNewID("filelog")

	logs := composeIndexName(component.KindReceiver, id, "logs")
	metrics := composeIndexName(component.KindReceiver, id, "metrics")

	assert.NotEqual(t, logs, metrics)
	assert.Contains(t, logs, "_logs")
	assert.Contains(t, metrics, "_metrics")
}

func TestComposeIndexName_KindDisambiguates(t *testing.T) {
	// Same component ID under different kinds must not collide. This
	// shouldn't happen in practice (collector component IDs are unique
	// per kind anyway) but the file_storage convention includes kind in
	// the name for safety, and we follow suit.
	id := component.MustNewID("foo")

	receiver := composeIndexName(component.KindReceiver, id, "")
	processor := composeIndexName(component.KindProcessor, id, "")

	assert.NotEqual(t, receiver, processor)
}

func TestKindString_AllKinds(t *testing.T) {
	cases := map[component.Kind]string{
		component.KindReceiver:  "receiver",
		component.KindProcessor: "processor",
		component.KindExporter:  "exporter",
		component.KindExtension: "extension",
		component.KindConnector: "connector",
	}
	for kind, want := range cases {
		assert.Equal(t, want, kindString(kind))
	}
	// An unknown kind value (the zero Kind) must fall back to a stable
	// string so the composed index name remains deterministic.
	assert.Equal(t, "other", kindString(component.Kind{}))
}
