// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package elasticsearchstorage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"go.opentelemetry.io/collector/component"

	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
)

// indexNamePrefix is the common prefix for storage indices. The legacy
// backend.Registry path uses the same prefix (see libbeat/statestore/backend/es/base.go),
// so OTel-managed indices and Beats-input-managed indices share a namespace.
const indexNamePrefix = "agentless-state-"

// indexMapping is the explicit mapping applied on first use of a storage
// index.
//
// The `v` field is declared as `object` with `enabled: false`: ES stores it
// verbatim in `_source` and does not parse, index, or type-check its
// contents. This:
//   - prevents dynamic-mapping conflicts when a consumer's value schema
//     evolves across releases (e.g. a field's type changes),
//   - avoids field-count growth for consumers with variable-shape values,
//   - costs nothing because we never search inside `v`.
//
// `updated_at` stays typed as `date` so operators can range-query for stale
// entries via the standard ES APIs.
var indexMapping = map[string]any{
	"settings": map[string]any{
		"number_of_shards":   1,
		"number_of_replicas": 0,
	},
	"mappings": map[string]any{
		"properties": map[string]any{
			"v":          map[string]any{"type": "object", "enabled": false},
			"updated_at": map[string]any{"type": "date"},
		},
	},
}

// composeIndexName builds an ES index name from an OTel component identity,
// mirroring the file_storage extension's kind+type+name+storageName scheme
// so two consumers with the same component.ID but different kinds (or
// different per-signal storageName) get distinct indices. Without this, a
// processor and a receiver named "foo" would collide on the same index.
//
// The composed name is then sanitized for ES index-naming rules.
func composeIndexName(kind component.Kind, id component.ID, storageName string) string {
	parts := []string{kindString(kind), id.Type().String(), id.Name()}
	if storageName != "" {
		parts = append(parts, storageName)
	}
	return indexNamePrefix + sanitizeIndexSuffix(strings.Join(parts, "_"))
}

// sanitizeIndexSuffix returns a string safe to use as the suffix of an ES
// index name.
//
// ES index naming rules: lowercase only; cannot contain \ / * ? " < > | , # :
// or whitespace; cannot start with -, _, or +; max 255 bytes. component.ID
// for a named instance like "akamai_siem/raw" stringifies with a forward
// slash, so the previous behaviour of feeding component.ID.String() into
// the index name produced invalid names like
// "agentless-state-akamai_siem/raw" — rejected by ES on first write.
//
// Any disallowed character is replaced with '-'. If the result would
// exceed 200 bytes (leaving room for the prefix to stay within the
// 255-byte ES limit), it is hash-truncated.
func sanitizeIndexSuffix(s string) string {
	s = strings.ToLower(s)

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= '0' && r <= '9',
			r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := strings.TrimLeft(b.String(), "-_+")

	if out == "" {
		// All input characters were illegal or leading-stripped. Fall back
		// to a hash of the original input so two distinct degenerate
		// inputs still land in distinct indices. component.ID enforces
		// alphanumeric+underscore, so this branch is defensive rather
		// than expected to fire.
		sum := sha256.Sum256([]byte(s))
		return "h" + hex.EncodeToString(sum[:8])
	}

	const maxSuffix = 200
	if len(out) > maxSuffix {
		sum := sha256.Sum256([]byte(out))
		// Keep a recognisable prefix; append a short hash for uniqueness.
		out = out[:maxSuffix-17] + "-" + hex.EncodeToString(sum[:8])
	}
	return out
}

// kindString stringifies an OTel component.Kind for use in an index name.
// Mirrors the file_storage extension's mapping so users moving between
// storage backends see consistent identifiers.
func kindString(k component.Kind) string {
	switch k {
	case component.KindReceiver:
		return "receiver"
	case component.KindProcessor:
		return "processor"
	case component.KindExporter:
		return "exporter"
	case component.KindExtension:
		return "extension"
	case component.KindConnector:
		return "connector"
	default:
		// component.Kind is a closed set in upstream OTel; this branch is
		// defensive so the resulting index name remains deterministic if a
		// new kind is introduced.
		return "other"
	}
}

// ensureIndex creates the storage index with the fixed mapping if it does
// not already exist. The 400 "resource_already_exists_exception" response is
// treated as success, so concurrent GetClient calls for the same index are
// race-safe.
//
// The mutex is the extension-wide one. Every call that uses the shared
// *eslegclient.Connection must take it: eslegclient.Connection reuses an
// internal response buffer and is documented as not safe for concurrent
// use by multiple goroutines.
func ensureIndex(mu *sync.Mutex, conn *eslegclient.Connection, indexName string) error {
	mu.Lock()
	defer mu.Unlock()

	status, body, err := conn.Request("PUT", "/"+indexName, "", nil, indexMapping)
	if err == nil {
		return nil
	}
	// PUT /<index> returns 400 with "resource_already_exists_exception" when
	// the index is already there — treat as success so two GetClient calls
	// for the same id don't fight.
	if status == http.StatusBadRequest && strings.Contains(string(body), "resource_already_exists_exception") {
		return nil
	}
	return fmt.Errorf("creating storage index %q: %w", indexName, err)
}
