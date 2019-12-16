// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package crypto

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIO(t *testing.T) {
	t.Run("encode and decode with the right password", func(t *testing.T) {
		passwd := []byte("hello")
		msg := []byte("bonjour la famille")
		dest := new(bytes.Buffer)

		// Encode
		w, err := NewWriterWithDefaults(dest, passwd)
		require.NoError(t, err)

		n, err := w.Write(msg)
		require.NoError(t, err)
		require.Equal(t, len(msg), n)

		// Guard to make sure we have not the same bytes.
		require.True(t, bytes.Index(dest.Bytes(), msg) == -1)

		r, err := NewReaderWithDefaults(dest, passwd)
		require.NoError(t, err)

		content, err := ioutil.ReadAll(r)
		require.NoError(t, err)

		require.Equal(t, msg, content)
	})

	t.Run("Large single write", func(t *testing.T) {
		passwd := []byte("hello")
		msg, err := randomBytes(1327)

		require.NoError(t, err)
		dest := new(bytes.Buffer)

		// Encode
		w, err := NewWriterWithDefaults(dest, passwd)
		require.NoError(t, err)

		n, err := io.Copy(w, bytes.NewBuffer(msg))
		require.NoError(t, err)
		require.Equal(t, int64(len(msg)), n)

		// Guard to make sure we have not the same bytes.
		require.True(t, bytes.Index(dest.Bytes(), msg) == -1)

		r, err := NewReaderWithDefaults(dest, passwd)
		require.NoError(t, err)

		content, err := ioutil.ReadAll(r)
		require.NoError(t, err)

		require.Equal(t, msg, content)
	})

	t.Run("try to decode with the wrong password", func(t *testing.T) {
		passwd := []byte("hello")
		msg := []byte("bonjour la famille")
		dest := new(bytes.Buffer)

		// Encode
		w, err := NewWriterWithDefaults(dest, passwd)
		require.NoError(t, err)

		n, err := w.Write(msg)
		require.NoError(t, err)
		require.Equal(t, len(msg), n)

		// Guard to make sure we have not the same bytes.
		require.True(t, bytes.Index(dest.Bytes(), msg) == -1)

		r, err := NewReaderWithDefaults(dest, []byte("bad password"))
		require.NoError(t, err)

		_, err = ioutil.ReadAll(r)
		require.Error(t, err)
	})

	t.Run("Make sure that buffered IO works with the encoder", func(t *testing.T) {
		passwd := []byte("hello")
		msg, err := randomBytes(2048)
		require.NoError(t, err)
		dest := new(bytes.Buffer)

		// Encode
		w, err := NewWriterWithDefaults(dest, passwd)
		require.NoError(t, err)

		b := bufio.NewWriterSize(w, 100)
		n, err := b.Write(msg)
		require.NoError(t, err)
		require.Equal(t, 2048, n)
		// err = b.Flush() //force flush
		require.NoError(t, err)

		require.True(t, len(dest.Bytes()) > 0)

		// Guard to make sure we have not the same bytes.
		require.True(t, bytes.Index(dest.Bytes(), msg) == -1)

		r, err := NewReaderWithDefaults(dest, passwd)
		require.NoError(t, err)

		content, err := ioutil.ReadAll(r)
		require.NoError(t, err)

		require.Equal(t, msg, content)
	})

	t.Run("Make sure that buffered IO works with the decoder", func(t *testing.T) {
		passwd := []byte("hello")
		msg, err := randomBytes(2048)
		require.NoError(t, err)
		dest := new(bytes.Buffer)

		// Encode
		w, err := NewWriterWithDefaults(dest, passwd)
		require.NoError(t, err)

		n, err := w.Write(msg)
		require.NoError(t, err)
		require.True(t, n == 2048)

		// Guard to make sure we have not the same bytes.
		require.True(t, bytes.Index(dest.Bytes(), msg) == -1)

		r, err := NewReaderWithDefaults(dest, passwd)
		require.NoError(t, err)

		b := bufio.NewReaderSize(r, 100)

		content, err := ioutil.ReadAll(b)
		require.NoError(t, err)

		require.Equal(t, msg, content)
	})

	t.Run("Missing explicit version", func(t *testing.T) {
		raw, err := randomBytes(2048)
		c := bytes.NewBuffer(raw)

		r, err := NewReaderWithDefaults(c, []byte("bad password"))
		require.NoError(t, err)

		b := bufio.NewReaderSize(r, 100)

		_, err = ioutil.ReadAll(b)
		require.Error(t, err)
	})

	t.Run("works with multiple writes", func(t *testing.T) {
		passwd := []byte("hello")

		expected := []byte("hello world bonjour la famille")

		dest := new(bytes.Buffer)

		// Encode
		w, err := NewWriterWithDefaults(dest, passwd)
		require.NoError(t, err)

		n, err := w.Write([]byte("hello world"))
		require.NoError(t, err)
		require.Equal(t, 11, n)

		n, err = w.Write([]byte(" bonjour la famille"))
		require.NoError(t, err)
		require.Equal(t, 19, n)

		// Guard to make sure we have not the same bytes.
		require.True(t, bytes.Index(dest.Bytes(), expected) == -1)

		r, err := NewReaderWithDefaults(dest, passwd)
		require.NoError(t, err)

		content, err := ioutil.ReadAll(r)
		require.NoError(t, err)

		require.Equal(t, expected, content)
	})
}
