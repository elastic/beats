// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/elastic/beats/v7/x-pack/libbeat/reader/parquet"
)

const (
	codecParquet = "parquet"
)

// decoder is an interface for decoding data from an io reader.
type decoder interface {
	// decode reads and decodes data from an io reader based on the codec type.
	decode() error
}

// newDecoder creates a new decoder based on the codec type.
// It returns a decoder type and an error if the codec type is not supported.
// If the reader config codec option is not set, it returns a nil decoder and nil error.
func newDecoder(p *s3ObjectProcessor, r io.Reader) (decoder, error) {
	switch p.readerConfig.Decoding.Codec {
	case "":
		return nil, nil
	case codecParquet:
		return newParquetDecoder(p, r)
	default:
		return nil, fmt.Errorf("unsupported codec type: %s", p.readerConfig.Decoding.Codec)
	}
}

// parquetDecoder is a decoder for parquet data.
type parquetDecoder struct {
	p      *s3ObjectProcessor
	reader *parquet.BufferedReader
}

// newParquetDecoder creates a new parquet decoder. It uses the libbeat parquet reader under the hood.
// It returns an error if the parquet reader cannot be created.
func newParquetDecoder(p *s3ObjectProcessor, r io.Reader) (decoder, error) {
	reader, err := parquet.NewBufferedReader(r, &parquet.Config{
		ProcessParallel: p.readerConfig.Decoding.Parquet.ProcessParallel,
		BatchSize:       p.readerConfig.Decoding.Parquet.BatchSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create parquet decoder: %w", err)
	}
	return &parquetDecoder{
		p:      p,
		reader: reader,
	}, nil
}

// decode reads and decodes a parquet data stream. After reading the parquet data it decodes
// the output to JSON and sends the decoded data to the readJSONSlice method to
// process further as individual JSON objects.
func (pd *parquetDecoder) decode() error {
	// releases the reader resources once processing is done
	defer pd.reader.Close()

	for pd.reader.Next() {
		data, err := pd.reader.Record()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("failed to read records from parquet record reader: %w", err)
		}

		err = pd.p.readJSONSlice(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("failed to read JSON data from arrow record: %w", err)
		}
	}

	return nil
}
