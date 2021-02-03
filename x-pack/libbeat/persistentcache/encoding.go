// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package persistentcache

import (
	"bytes"
	"encoding/json"

	ugorjicodec "github.com/ugorji/go/codec"
)

type codec interface {
	Decode([]byte, interface{}) error
	Encode(interface{}) ([]byte, error)
}

type jsonCodec struct{}

func newJSONCodec() *jsonCodec {
	return &jsonCodec{}
}

// Encode encodes an object in json format.
func (*jsonCodec) Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Decode decodes an object from its json representation.
func (*jsonCodec) Decode(d []byte, v interface{}) error {
	return json.Unmarshal(d, v)
}

type cborCodec struct {
	handle ugorjicodec.CborHandle
}

func newCBORCodec() *cborCodec {
	return &cborCodec{}
}

// Encode encodes an object in cbor format.
func (c *cborCodec) Encode(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := ugorjicodec.NewEncoder(&buf, &c.handle)
	err := enc.Encode(v)
	return buf.Bytes(), err
}

// Decode decodes an object from its cbor representation.
func (c *cborCodec) Decode(d []byte, v interface{}) error {
	dec := ugorjicodec.NewDecoder(bytes.NewReader(d), &c.handle)
	return dec.Decode(v)
}
