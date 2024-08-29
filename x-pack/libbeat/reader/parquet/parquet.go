// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package parquet

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/apache/arrow/go/v14/parquet"
	"github.com/apache/arrow/go/v14/parquet/file"
	"github.com/apache/arrow/go/v14/parquet/pqarrow"

	"github.com/elastic/elastic-agent-libs/logp"
)

// BufferedReader parses parquet inputs from io streams.
type BufferedReader struct {
	cfg          *Config
	fileReader   *file.Reader
	recordReader pqarrow.RecordReader
	log          *logp.Logger
}

// NewBufferedReader creates a new reader that can decode parquet data from an io.Reader.
// It will return an error if the parquet data stream cannot be read.
// Note: As io.ReadAll is used, the entire data stream would be read into memory, so very large data streams
// may cause memory bottleneck issues.
func NewBufferedReader(r io.Reader, cfg *Config) (*BufferedReader, error) {
	log := logp.L().Named("reader.parquet")
	start := time.Now()

	if cfg.BatchSize == 0 {
		cfg.BatchSize = 1
	}
	log.Debugw("creating parquet reader", "batch_size", cfg.BatchSize)

	// reads the contents of the reader object into a byte slice
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read data from stream reader: %w", err)
	}
	log.Debugw("read data from stream reader", "size", len(data), "duration", fmt.Sprintf("%v", time.Since(start)))

	// defines a memory allocator for allocating memory for Arrow objects
	pool := memory.NewCheckedAllocator(&memory.GoAllocator{})
	// constructs a parquet file reader object from the byte slice data
	pf, err := file.NewParquetReader(bytes.NewReader(data), file.WithReadProps(parquet.NewReaderProperties(pool)))
	if err != nil {
		return nil, fmt.Errorf("failed to create parquet reader: %w", err)
	}
	log.Debugw("created parquet reader", "duration", fmt.Sprintf("%v", time.Since(start)))

	// constructs a reader for converting to Arrow objects from an existing parquet file reader object
	reader, err := pqarrow.NewFileReader(pf, pqarrow.ArrowReadProperties{
		Parallel:  cfg.ProcessParallel,
		BatchSize: int64(cfg.BatchSize),
	}, pool)
	if err != nil {
		return nil, fmt.Errorf("failed to create pqarrow parquet reader: %w", err)
	}
	log.Debugw("created pqarrow parquet reader", "duration", fmt.Sprintf("%v", time.Since(start)))

	// constructs a record reader that is capable of reding entire sets of arrow records
	rr, err := reader.GetRecordReader(context.Background(), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create parquet record reader: %w", err)
	}
	log.Debugw("initialization process completed", "duration", fmt.Sprintf("%v", time.Since(start)))

	return &BufferedReader{
		cfg:          cfg,
		recordReader: rr,
		fileReader:   pf,
		log:          log,
	}, nil
}

// Next advances the pointer to point to the next record and returns true if the next record exists.
// It will return false if there are no more records to read.
func (sr *BufferedReader) Next() bool {
	next := sr.recordReader.Next()
	if !next {
		sr.log.Debugw("no more records to read", "next", next)
	}
	return next
}

// Record reads the current record from the parquet file and returns it as a JSON marshaled byte slice.
// If no more records are available, the []byte slice will be nil and io.EOF will be returned as an error.
// A JSON marshal error will be returned if the record cannot be marshalled.
func (sr *BufferedReader) Record() ([]byte, error) {
	rec := sr.recordReader.Record()
	if rec == nil {
		sr.log.Debugw("reached the end of the record reader", "record_reader", rec)
		return nil, io.EOF
	}
	defer rec.Release()
	val, err := rec.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON for parquet value: %w", err)
	}
	sr.log.Debugw("records successfully read", "batch_size", sr.cfg.BatchSize)

	return val, nil
}

// Close closes the stream reader and releases all resources.
// It will return an error if the fileReader fails to close.
func (sr *BufferedReader) Close() error {
	sr.recordReader.Release()
	return sr.fileReader.Close()
}
