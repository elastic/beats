// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package parquet

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/apache/arrow/go/arrow/memory"
	"github.com/apache/arrow/go/v11/parquet"
	"github.com/apache/arrow/go/v11/parquet/file"
	"github.com/apache/arrow/go/v11/parquet/pqarrow"
)

// StreamReader parses parquet inputs from io streams.
type StreamReader struct {
	cfg          *Config
	fileReader   *file.Reader
	recordReader pqarrow.RecordReader
}

// NewStreamReader creates a new reader that can decode parquet data from an io.Reader.
func NewStreamReader(r io.Reader, cfg *Config) (*StreamReader, error) {
	batchSize := 1
	if cfg.BatchSize > 1 {
		batchSize = cfg.BatchSize
	}

	// reads the contents of the reader object into a byte array
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to data from stream reader: %w", err)
	}

	// defines a memory allocator for allocating memory for Arrow objects
	pool := memory.NewCheckedAllocator(memory.DefaultAllocator)

	pf, err := file.NewParquetReader(bytes.NewReader(data), file.WithReadProps(parquet.NewReaderProperties(pool)))
	if err != nil {
		return nil, fmt.Errorf("failed to create parquet reader: %w", err)
	}

	// constructs a reader for converting to Arrow objects from an existing parquet file reader object
	reader, err := pqarrow.NewFileReader(pf, pqarrow.ArrowReadProperties{
		Parallel:  cfg.ProcessParallel,
		BatchSize: int64(batchSize),
	}, pool)
	if err != nil {
		return nil, fmt.Errorf("failed to create pqarrow parquet reader: %w", err)
	}

	// constructs a record reader that is capable of reding entire sets of arrow records
	rr, err := reader.GetRecordReader(context.Background(), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create parquet record reader: %w", err)
	}

	return &StreamReader{
		cfg:          cfg,
		recordReader: rr,
		fileReader:   pf,
	}, nil
}

// Next returns true if there are more records to read.
// It will return false if there are no more records to read.
func (sr *StreamReader) Next() bool {
	return sr.recordReader.Next()
}

// Record reads the current record from the parquet file and returns it as a json marshaled byte array.
// If no more records are available, the []byte array will be nil. It will return
// an error if the record could not be marshalled.
func (sr *StreamReader) Record() ([]byte, error) {
	var val []byte
	var err error
	rec := sr.recordReader.Record()
	if rec != nil {
		defer rec.Release()
		val, err = rec.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON for parquet value: %w", err)
		}
	}

	return val, nil
}

// Close closes the stream reader and releases all resources.
// It will return an error if the fileReader fails to close.
func (sr *StreamReader) Close() error {
	sr.recordReader.Release()
	return sr.fileReader.Close()
}
