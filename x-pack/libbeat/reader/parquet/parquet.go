// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package parquet

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/apache/arrow/go/arrow/memory"
	"github.com/apache/arrow/go/v11/parquet"
	"github.com/apache/arrow/go/v11/parquet/file"
	"github.com/apache/arrow/go/v11/parquet/pqarrow"
)

// StreamReader parses parquet inputs from io streams
type StreamReader struct {
	cfg          *Config
	fileReader   *file.Reader
	recordReader pqarrow.RecordReader
}

// NewStreamReader creates a new reader that can decode parquet.
func NewStreamReader(r io.Reader, cfg *Config) (*StreamReader, error) {
	batchSize := 1
	if cfg.BatchSize > 1 {
		batchSize = cfg.BatchSize
	}

	// read the contents of the reader object
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to data from stream reader: %w", err)
	}

	// define a memory allocator
	pool := memory.NewCheckedAllocator(memory.DefaultAllocator)

	pf, err := file.NewParquetReader(bytes.NewReader(data), file.WithReadProps(parquet.NewReaderProperties(pool)))
	if err != nil {
		return nil, fmt.Errorf("failed to create parquet reader: %w", err)
	}

	// constructs a reader for converting to Arrow objects from an existing parquet file reader object.
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

// Next returns true if there are more records to read
func (sr *StreamReader) Next() bool {
	return sr.recordReader.Next()
}

// Read reads the next record from the parquet file
func (sr *StreamReader) Read() ([]byte, error) {
	var val []byte
	rec, err := sr.recordReader.Read()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("failed to read records from parquet record reader: %w", err)
	}
	if rec != nil {
		defer rec.Release()
		val, err = rec.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON for parquet value: %w", err)
		}
	}

	return val, nil
}

// Close closes the reader
func (sr *StreamReader) Close() error {
	sr.recordReader.Release()
	return sr.fileReader.Close()
}
