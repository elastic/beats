// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"fmt"
	"io"

	"github.com/elastic/beats/v7/x-pack/libbeat/reader/parquet"
)

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
