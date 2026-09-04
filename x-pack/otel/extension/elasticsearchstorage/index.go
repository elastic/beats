// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchstorage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"go.opentelemetry.io/collector/component"
)

// indexNamePrefix is the common prefix for storage indices. The legacy
// backend.Registry path uses the same prefix (see libbeat/statestore/backend/es/base.go),
// so OTel-managed indices and Beats-input-managed indices share a namespace.
const indexNamePrefix = "agentless-state-"

// indexMappings is the explicit field mapping applied on first use of a
// storage index.
//
// The `v` field is declared as `object` with `enabled: false`: ES stores it
// verbatim in `_source` and does not parse, index, or type-check its
// contents. This:
//   - prevents dynamic-mapping conflicts when a consumer's value schema
//     evolves across releases (e.g. a field's type changes),
//   - avoids field-count growth for consumers with variable-shape values,
//   - costs nothing because we never search inside `v`.
//
// `enc` records how `v` was encoded ("json" for verbatim JSON, "base64" for
// opaque bytes) so Get can reverse it; it is a `keyword` because it is a small
// closed set never used for full-text search. `updated_at` stays typed as
// `date` so operators can range-query for stale entries via the standard ES
// APIs.
var indexMappings = map[string]any{
	"properties": map[string]any{
		"v":          map[string]any{"type": "object", "enabled": false},
		"enc":        map[string]any{"type": "keyword"},
		"updated_at": map[string]any{"type": "date"},
	},
}

// Default index layout for a tiny state index: single shard, no replica. Both
// are configurable via IndexConfig; a non-positive value falls back to these.
const (
	defaultNumberOfShards   = 1
	defaultNumberOfReplicas = 0
)

// indexCreateBody builds the create-index request body. The field mappings are
// always included; the shard/replica settings are included only for stateful
// clusters, since Elastic Cloud Serverless manages them itself and rejects
// them on create. Shard/replica counts come from idx, defaulting when unset.
func indexCreateBody(serverless bool, idx IndexConfig) map[string]any {
	body := map[string]any{"mappings": indexMappings}
	if serverless {
		return body
	}

	shards := idx.NumberOfShards
	if shards <= 0 {
		shards = defaultNumberOfShards
	}
	replicas := idx.NumberOfReplicas
	if replicas < 0 {
		replicas = defaultNumberOfReplicas
	}
	body["settings"] = map[string]any{
		"number_of_shards":   shards,
		"number_of_replicas": replicas,
	}
	return body
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
		case 'a' <= r && r <= 'z',
			'0' <= r && r <= '9',
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

// createIndex creates the client's storage index with the fixed mapping if
// it does not already exist. The 400 "resource_already_exists_exception"
// response is treated as success, so concurrent creators for the same index
// are race-safe. The shard/replica settings are dropped on serverless, where
// they are not permitted (see indexCreateBody).
//
// The request goes through the client's retrying transport, so a transient
// failure on index creation is retried the same way as a document write.
func (c *esStorageClient) createIndex(ctx context.Context) error {
	serverless, err := c.ext.isServerless()
	if err != nil {
		return err
	}
	status, body, err := c.request(ctx, "PUT", "/"+c.index, nil, indexCreateBody(serverless, c.ext.cfg.Index))
	if err == nil {
		return nil
	}
	// PUT /<index> returns 400 with "resource_already_exists_exception" when
	// the index is already there — treat as success so two creators for the
	// same identity don't fight.
	if status == http.StatusBadRequest && strings.Contains(string(body), "resource_already_exists_exception") {
		return nil
	}
	return fmt.Errorf("creating storage index %q: %w", c.index, err)
}
