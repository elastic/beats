// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"bytes"
	"errors"
	"io"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountReader(t *testing.T) {
	t.Run("byte_counting", func(t *testing.T) {
		data := []byte("hello world")
		var inFlight atomic.Int64

		r := newCountReader(io.NopCloser(bytes.NewReader(data)), &inFlight, 0)

		buf := make([]byte, 5)
		n, err := r.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, int64(5), inFlight.Load())
		assert.Equal(t, int64(5), r.read)

		n, err = r.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, int64(10), inFlight.Load())
		assert.Equal(t, int64(10), r.read)

		n, err = r.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, 1, n) // "d" remains.
		assert.Equal(t, int64(11), inFlight.Load())
		assert.Equal(t, int64(11), r.read)

		_, err = r.Read(buf)
		assert.Equal(t, io.EOF, err)
		assert.Equal(t, int64(11), inFlight.Load())
	})

	t.Run("close", func(t *testing.T) {
		data := []byte("hello world")
		var inFlight atomic.Int64
		inFlight.Store(100) // Pre-existing in-flight bytes.

		r := newCountReader(io.NopCloser(bytes.NewReader(data)), &inFlight, 0)

		// Read all data.
		buf := make([]byte, 20)
		n, err := r.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, 11, n)
		assert.Equal(t, int64(111), inFlight.Load()) // 100 + 11

		// Close should subtract the bytes read.
		err = r.Close()
		require.NoError(t, err)
		assert.Equal(t, int64(100), inFlight.Load()) // Back to original.

		// Double close should be safe.
		err = r.Close()
		require.NoError(t, err)
		assert.Equal(t, int64(100), inFlight.Load())
	})

	t.Run("exceed_max", func(t *testing.T) {
		data := []byte("hello world hello world") // 23 bytes
		var inFlight atomic.Int64

		r := newCountReader(io.NopCloser(bytes.NewReader(data)), &inFlight, 15)

		buf := make([]byte, 10)

		// First read: 10 bytes, under limit
		n, err := r.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, 10, n)
		assert.Equal(t, int64(10), inFlight.Load())

		// Second read: would push us to 20 bytes, over limit of 15.
		n, err = r.Read(buf)
		assert.Equal(t, 10, n) // Still returns the bytes read.
		assert.True(t, errors.Is(err, errMaxInFlightExceeded))
		assert.Equal(t, int64(20), inFlight.Load())

		// Close still subtracts our bytes.
		err = r.Close()
		require.NoError(t, err)
		assert.Equal(t, int64(0), inFlight.Load())
	})

	t.Run("exceed_max_with_pre_existing", func(t *testing.T) {
		data := []byte("hello") // 5 bytes.
		var inFlight atomic.Int64
		inFlight.Store(12) // Pre-existing in-flight bytes.

		// Max is 15, so 12 + 5 = 17 will exceed
		r := newCountReader(io.NopCloser(bytes.NewReader(data)), &inFlight, 15)

		buf := make([]byte, 10)
		n, err := r.Read(buf)
		assert.Equal(t, 5, n)
		assert.True(t, errors.Is(err, errMaxInFlightExceeded))
		assert.Equal(t, int64(17), inFlight.Load())

		// Close subtracts only our bytes.
		err = r.Close()
		require.NoError(t, err)
		assert.Equal(t, int64(12), inFlight.Load()) // Back to pre-existing.
	})

	t.Run("no_limit", func(t *testing.T) {
		data := make([]byte, 1000)
		var inFlight atomic.Int64

		// max=0 means no limit
		r := newCountReader(io.NopCloser(bytes.NewReader(data)), &inFlight, 0)

		buf := make([]byte, 1000)
		n, err := r.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, 1000, n)
		assert.Equal(t, int64(1000), inFlight.Load())

		err = r.Close()
		require.NoError(t, err)
		assert.Equal(t, int64(0), inFlight.Load())
	})
}
