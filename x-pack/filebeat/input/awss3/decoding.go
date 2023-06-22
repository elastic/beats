// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"fmt"
	"io"

	"github.com/elastic/beats/v7/x-pack/libbeat/reader/parquet"
)

// decoder is an interface for decoding data from an io reader.
type decoder interface {
	// decode reads and decodes data from an io reader based on the codec type.
	// It returns the decoded data and an error if the data cannot be decoded.
	decode() ([]byte, error)
	// next advances the decoder to the next data item and returns true if there is more data to be decoded.
	next() bool
	// close closes the decoder and releases any resources associated with it.
	// It returns an error if the decoder cannot be closed.
	close() error
}

// newDecoder creates a new decoder based on the codec type.
// It returns a decoder type and an error if the codec type is not supported.
// If the reader config codec option is not set, it returns a nil decoder and nil error.
func newDecoder(config decoderConfig, r io.Reader) (decoder, error) {
	switch {
	case config.Codec == nil:
		return nil, nil
	case config.Codec.Parquet != nil:
		return newParquetDecoder(config, r)
	default:
		return nil, fmt.Errorf("unsupported config value: %v", config)
	}
}

// parquetDecoder is a decoder for parquet data.
type parquetDecoder struct {
	reader *parquet.BufferedReader
}

// newParquetDecoder creates a new parquet decoder. It uses the libbeat parquet reader under the hood.
// It returns an error if the parquet reader cannot be created.
func newParquetDecoder(config decoderConfig, r io.Reader) (decoder, error) {
	reader, err := parquet.NewBufferedReader(r, &parquet.Config{
		ProcessParallel: config.Codec.Parquet.ProcessParallel,
		BatchSize:       config.Codec.Parquet.BatchSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create parquet decoder: %w", err)
	}
	return &parquetDecoder{
		reader: reader,
	}, nil
}

// next advances the parquet decoder to the next data item and returns true if there is more data to be decoded.
func (pd *parquetDecoder) next() bool {
	return pd.reader.Next()
}

// decode reads and decodes a parquet data stream. After reading the parquet data it decodes
// the output to JSON and returns it as a byte slice. It returns an error if the data cannot be decoded.
func (pd *parquetDecoder) decode() ([]byte, error) {
	data, err := pd.reader.Record()
	if err != nil {
		return nil, err
	}
	return data, nil
}

// close closes the parquet decoder and releases the resources.
func (pd *parquetDecoder) close() error {
	return pd.reader.Close()
}
