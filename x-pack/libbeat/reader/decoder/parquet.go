// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package decoder

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
func NewParquetDecoder(config ParquetCodecConfig, r io.Reader) (Decoder, error) {
	reader, err := parquet.NewBufferedReader(r, &parquet.Config{
		ProcessParallel: config.ProcessParallel,
		BatchSize:       config.BatchSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create parquet decoder: %w", err)
	}
	return &parquetDecoder{
		reader: reader,
	}, nil
}

// next advances the parquet decoder to the next data item and returns true if there is more data to be decoded.
func (pd *parquetDecoder) More() bool {
	// cache the results of Next() locally
	// set a flag to state we have prepped a
	return pd.Next()
}

// next advances the parquet decoder to the next data item and returns true if there is more data to be decoded.
func (pd *parquetDecoder) Next() bool {
	// update a boolean
	return pd.reader.Next()
}

// decode reads and decodes a parquet data stream. After reading the parquet data it decodes
// the output to JSON and returns it as a byte slice. It returns an error if the data cannot be decoded.
func (pd *parquetDecoder) Decode() ([]byte, error) {
	data, err := pd.reader.Record()
	if err != nil {
		return nil, err
	}
	return data, nil
}

// close closes the parquet decoder and releases the resources.
func (pd *parquetDecoder) Close() error {
	return pd.reader.Close()
}
