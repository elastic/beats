// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

import (
	"fmt"
	"io"
)

// decoder is an interface for decoding data from an io.Reader.
type decoder interface {
	// decode reads and decodes data from an io reader based on the codec type.
	// It returns the decoded data and an error if the data cannot be decoded.
	decode() ([]byte, error)
	// next advances the decoder to the next data item and returns true if there is more data to be decoded.
	next() bool
	// close closes the decoder and releases any resources associated with it.
	// It returns an error if the decoder cannot be closed.

	// more returns whether there are more records to read.
	more() bool

	close() error
}

// valueDecoder is a decoder that can decode directly to a JSON serialisable value.
type valueDecoder interface {
	decoder

	decodeValue() ([]byte, map[string]any, error)
}

// newDecoder creates a new decoder based on the codec type.
// It returns a decoder type and an error if the codec type is not supported.
// If the reader config codec option is not set, it returns a nil decoder and nil error.
func newDecoder(cfg decoderConfig, r io.Reader) (decoder, error) {
	switch {
	case cfg.Codec == nil:
		return nil, nil
	case cfg.Codec.CSV != nil:
		return newCSVDecoder(cfg, r)
	default:
		return nil, fmt.Errorf("unsupported config value: %v", cfg)
	}
}
