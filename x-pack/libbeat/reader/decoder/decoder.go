// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package decoder

import (
	"fmt"
	"io"
)

// decoder is an interface for decoding data from an io.Reader.
type Decoder interface {
	// decode reads and decodes data from an io reader based on the codec type.
	// It returns the decoded data and an error if the data cannot be decoded.
	Decode() ([]byte, error)
	// next advances the decoder to the next data item and returns true if there is more data to be decoded.
	Next() bool
	// close closes the decoder and releases any resources associated with it.
	// It returns an error if the decoder cannot be closed.
	Close() error
	// more returns whether there are more records to read.
	More() bool
}

// valueDecoder is a decoder that can decode directly to a JSON serialisable value.
type ValueDecoder interface {
	Decoder

	DecodeValue() (int64, []byte, map[string]any, error)
}

// newDecoder creates a new decoder based on the codec type.
// It returns a decoder type and an error if the codec type is not supported.
// If the reader config codec option is not set, it returns a nil decoder and nil error.
func NewDecoder(cfg Config, r io.Reader) (Decoder, error) {
	codec := cfg.Codec

	if cfg.Codec == nil {
		return nil, nil
	} else if err := codec.Validate(); err != nil {
		return nil, err
	}

	var result Decoder
	switch {
	case cfg.Codec.CSV != nil:
		csv := codec.CSV
		result, _ = NewCSVDecoder(*csv, r)
	case cfg.Codec.Parquet != nil:
		pqt := codec.Parquet
		result, _ = NewParquetDecoder(*pqt, r)
	default:
		return nil, fmt.Errorf("unsupported config value: %v", cfg)
	}

	return result, nil
}
