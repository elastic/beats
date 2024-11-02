// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"compress/gzip"
	"errors"
	"io"
	"sync"
)

var gzipDecoderPool = sync.Pool{
	New: func() interface{} {
		return new(gzip.Reader)
	},
}

type pooledGzipReader struct {
	Reader *gzip.Reader
	closer io.Closer
}

func newPooledGzipReader(r io.ReadCloser) (*pooledGzipReader, error) {
	gzipReader := gzipDecoderPool.Get().(*gzip.Reader)
	if err := gzipReader.Reset(r); err != nil {
		gzipDecoderPool.Put(gzipReader)
		return nil, err
	}
	return &pooledGzipReader{Reader: gzipReader, closer: r}, nil
}

// Read implements io.Reader, reading uncompressed bytes from its underlying Reader.
func (r *pooledGzipReader) Read(b []byte) (int, error) {
	return r.Reader.Read(b)
}

// Close closes the Reader and the underlying source.
// In order for the GZIP checksum to be verified, the reader must be
// fully consumed until the io.EOF.
//
// After this call the reader should not be reused because it is returned to the pool.
func (r *pooledGzipReader) Close() error {
	err := r.Reader.Close()
	err = errors.Join(err, r.closer.Close())
	gzipDecoderPool.Put(r.Reader)
	r.Reader = nil
	return err
}
