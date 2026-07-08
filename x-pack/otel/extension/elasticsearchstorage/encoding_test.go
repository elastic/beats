// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package elasticsearchstorage

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roundTrip encodes value with mode then decodes it back, returning the
// recovered bytes and the encoding tag that was stored.
func roundTrip(t *testing.T, value []byte, mode string) ([]byte, string) {
	t.Helper()
	raw, enc := encodeValue(value, mode)
	got, err := decodeValue(raw, enc)
	require.NoError(t, err, "decodeValue must not fail for a value we just encoded")
	return got, enc
}

func TestEncodeDecode_RoundTrip(t *testing.T) {
	cases := []struct {
		name    string
		value   []byte
		mode    string
		wantEnc string
	}{
		{"json object auto", []byte(`{"a":1,"b":[1,2,3]}`), "auto", encJSON},
		{"json object empty mode", []byte(`{"a":1}`), "", encJSON},
		{"bare number auto", []byte(`42`), "auto", encJSON},
		{"bare string auto", []byte(`"hello"`), "auto", encJSON},
		{"non-json auto -> base64", []byte{0x00, 0x01, 0xff, 0xfe}, "auto", encBase64},
		{"json pinned", []byte(`{"a":1}`), "json", encJSON},
		{"bytes pinned even for json", []byte(`{"a":1}`), "bytes", encBase64},
		{"bytes pinned for raw", []byte{0x00, 0xff}, "bytes", encBase64},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, enc := roundTrip(t, tc.value, tc.mode)
			assert.Equal(t, tc.wantEnc, enc, "encoding tag")
			assert.Equal(t, tc.value, got, "value must round-trip byte-for-byte")
		})
	}
}

func TestEncodeValue_JSONVerbatim_NoReserialization(t *testing.T) {
	// The json branch must store the caller's bytes unchanged — key order
	// and whitespace preserved — never decode-then-re-encode.
	in := []byte(`{"z":1,   "a":2}`)
	raw, enc := encodeValue(in, "auto")
	assert.Equal(t, encJSON, enc)
	assert.Equal(t, string(in), string(raw), "json payload must be stored verbatim")
}

func TestEncodeValue_LargeInt64_PreservedExactly(t *testing.T) {
	// A value > 2^53 would lose precision through a float64 round-trip. The
	// verbatim json branch avoids that.
	const large int64 = 9_000_000_000_000_000_001
	in := []byte(fmt.Sprintf(`{"big":%d}`, large))
	got, enc := roundTrip(t, in, "auto")
	assert.Equal(t, encJSON, enc)
	assert.Equal(t, in, got)
}

func TestDecodeValue_MissingEncTreatedAsJSON(t *testing.T) {
	// Documents written before enc existed have no enc field; they must read
	// back as JSON (forward compatibility).
	got, err := decodeValue([]byte(`{"a":1}`), "")
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":1}`, string(got))
}

func TestDecodeValue_EmptyValue(t *testing.T) {
	got, err := decodeValue(nil, encJSON)
	require.NoError(t, err)
	assert.Nil(t, got, "empty v must decode to (nil, nil)")
}

func TestDecodeValue_UnknownEncoding(t *testing.T) {
	_, err := decodeValue([]byte(`"x"`), "gzip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown value encoding")
}

func TestDecodeValue_BadBase64(t *testing.T) {
	_, err := decodeValue([]byte(`"not*valid*base64"`), encBase64)
	require.Error(t, err)
}
