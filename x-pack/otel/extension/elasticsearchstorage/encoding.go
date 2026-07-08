// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package elasticsearchstorage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// Value encodings recorded in the document's `enc` field.
const (
	encJSON   = "json"
	encBase64 = "base64"
)

// docBody is the wire format the client writes and reads.
//
// Field tags use encoding/json (not the eslegclient go-structform "struct"
// tag) because the body is marshaled with stdlib json and submitted via
// eslegclient.RawEncoding, which bypasses go-structform. Marshaling the
// caller's value verbatim as json.RawMessage guarantees the bytes ES sees
// under `v` are exactly the bytes the caller passed to Set (for the JSON
// encoding), preserving numeric precision and key order.
type docBody struct {
	V         json.RawMessage `json:"v"`
	Enc       string          `json:"enc"`
	UpdatedAt string          `json:"updated_at"`
}

// encodeValue turns the caller's bytes into the `v` payload and its encoding
// tag according to mode.
//
// Round-trip invariant: the JSON branch stores the caller's bytes unchanged —
// never decode-then-re-encode — so an opaque payload that happens to be valid
// JSON (e.g. a bare number) comes back byte-identical.
func encodeValue(value []byte, mode string) (json.RawMessage, string) {
	switch mode {
	case encJSON:
		return json.RawMessage(value), encJSON
	case "bytes":
		return base64Envelope(value), encBase64
	default: // "auto" or ""
		if json.Valid(value) {
			return json.RawMessage(value), encJSON
		}
		return base64Envelope(value), encBase64
	}
}

// base64Envelope base64-encodes value and returns it as a JSON string.
func base64Envelope(value []byte) json.RawMessage {
	// Marshaling a string cannot fail.
	q, _ := json.Marshal(base64.StdEncoding.EncodeToString(value))
	return json.RawMessage(q)
}

// decodeValue reverses encodeValue for a stored document. A missing or empty
// `v` yields (nil, nil). A missing enc is treated as JSON for forward
// compatibility with documents written before enc existed.
func decodeValue(v json.RawMessage, enc string) ([]byte, error) {
	if len(v) == 0 {
		return nil, nil
	}
	switch enc {
	case "", encJSON:
		// Copy so the returned bytes never alias a shared buffer.
		out := make([]byte, len(v))
		copy(out, v)
		return out, nil
	case encBase64:
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return nil, fmt.Errorf("decoding base64 envelope: %w", err)
		}
		decoded, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return nil, fmt.Errorf("decoding base64 value: %w", err)
		}
		return decoded, nil
	default:
		return nil, fmt.Errorf("unknown value encoding %q", enc)
	}
}
