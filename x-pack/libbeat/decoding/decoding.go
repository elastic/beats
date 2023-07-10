// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package decoding

import (
	"bufio"
	"fmt"
	"io"
)

// Decoder is an interface for decoding data from an io reader.
type Decoder interface {
	// Decode reads and decodes data from an io reader based on the codec type.
	// It returns the decoded data and an error if the data cannot be decoded.
	Decode() ([]byte, error)
	// Next advances the decoder to the next data item and returns true if there is more data to be decoded.
	Next() bool
	// Close closes the decoder and releases any resources associated with it.
	// It returns an error if the decoder cannot be closed.
	Close() error
	// Type returns the underlying type of the decoder.
	Type() interface{}
}

// NewDecoder creates a new decoder based on the codec type.
// It returns a decoder type and an error if the codec type is not supported.
// If the reader config codec option is not set, it returns a nil decoder and nil error.
func NewDecoder(config DecoderConfig, r io.Reader, offset int64) (Decoder, error) {
	// apply gzipdecoder if required
	var err error
	r, err = addGzipDecoderIfNeeded(bufio.NewReader(r))
	if err != nil {
		return nil, fmt.Errorf("failed to add gzip decoder to data stream with error: %w", err)
	}
	switch {
	case config.Codec == nil:
		return nil, nil
	case config.Codec.Parquet != nil:
		return NewParquetDecoder(config, r)
	case config.Codec.JSON != nil:
		return NewJSONDecoder(config, r)
	default:
		return nil, fmt.Errorf("unsupported config value: %v", config)
	}
}
