// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package storage

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestReplaceOrRollbackStore(t *testing.T) {
	in := bytes.NewReader([]byte{})

	replaceWith := []byte("new content")
	oldContent := []byte("old content")

	success := NewHandlerStore(func(_ io.Reader) error { return nil })
	failure := NewHandlerStore(func(_ io.Reader) error { return errors.New("fail") })

	t.Run("when the save is successful with target and source don't match", func(t *testing.T) {
		target, err := genFile(oldContent)
		require.NoError(t, err)
		dir := filepath.Dir(target)
		defer os.RemoveAll(dir)

		requireFilesCount(t, dir, 1)

		s := NewReplaceOnSuccessStore(
			target,
			replaceWith,
			success,
		)

		err = s.Save(in)
		require.NoError(t, err)

		writtenContent, err := ioutil.ReadFile(target)
		require.NoError(t, err)

		require.True(t, bytes.Equal(writtenContent, replaceWith))
		requireFilesCount(t, dir, 2)
	})

	t.Run("when save is not successful", func(t *testing.T) {
		target, err := genFile(oldContent)
		require.NoError(t, err)
		dir := filepath.Dir(target)
		defer os.RemoveAll(dir)

		requireFilesCount(t, dir, 1)

		s := NewReplaceOnSuccessStore(
			target,
			replaceWith,
			failure,
		)

		err = s.Save(in)
		require.Error(t, err)

		writtenContent, err := ioutil.ReadFile(target)
		require.NoError(t, err)

		require.True(t, bytes.Equal(writtenContent, oldContent))
		requireFilesCount(t, dir, 1)
	})

	t.Run("when save is successful with target and source content match", func(t *testing.T) {
		target, err := genFile(replaceWith)
		require.NoError(t, err)
		dir := filepath.Dir(target)
		defer os.RemoveAll(dir)

		requireFilesCount(t, dir, 1)

		s := NewReplaceOnSuccessStore(
			target,
			replaceWith,
			failure,
		)

		err = s.Save(in)
		require.Error(t, err)

		writtenContent, err := ioutil.ReadFile(target)
		require.NoError(t, err)

		require.True(t, bytes.Equal(writtenContent, replaceWith))
		requireFilesCount(t, dir, 1)
	})

	t.Run("when target file do not exist", func(t *testing.T) {
		s := NewReplaceOnSuccessStore(
			fmt.Sprintf("%s/%d", os.TempDir(), time.Now().Unix()),
			replaceWith,
			success,
		)
		err := s.Save(in)
		require.Error(t, err)
	})
}

func TestDiskStore(t *testing.T) {
	t.Run("when the target file already exists", func(t *testing.T) {
		target, err := genFile([]byte("hello world"))
		require.NoError(t, err)
		defer os.Remove(target)
		d := &DiskStore{target: target}

		msg := []byte("bonjour la famille")
		err = d.Save(bytes.NewReader(msg))
		require.NoError(t, err)

		content, err := ioutil.ReadFile(target)
		require.NoError(t, err)

		require.Equal(t, msg, content)
	})

	t.Run("when the target do no exist", func(t *testing.T) {
		dir, err := ioutil.TempDir("", "configs")
		require.NoError(t, err)
		defer os.Remove(dir)

		target := filepath.Join(dir, "hello.txt")
		d := &DiskStore{target: target}

		msg := []byte("bonjour la famille")
		err = d.Save(bytes.NewReader(msg))
		require.NoError(t, err)

		content, err := ioutil.ReadFile(target)
		require.NoError(t, err)

		require.Equal(t, msg, content)
	})

	t.Run("return an io.ReadCloser to the target file", func(t *testing.T) {
		msg := []byte("bonjour la famille")
		target, err := genFile(msg)
		require.NoError(t, err)

		d := &DiskStore{target: target}
		r, err := d.Load()
		require.NoError(t, err)
		defer r.Close()

		content, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, msg, content)
	})
}

func TestEncryptedDiskStore(t *testing.T) {
	t.Run("when the target file already exists", func(t *testing.T) {
		target, err := genFile([]byte("hello world"))
		require.NoError(t, err)
		defer os.Remove(target)
		d := &EncryptedDiskStore{target: target}

		msg := []byte("bonjour la famille")
		err = d.Save(bytes.NewReader(msg))
		require.NoError(t, err)

		// lets read the file
		nd := &EncryptedDiskStore{target: target}
		r, err := nd.Load()
		require.NoError(t, err)

		content, err := ioutil.ReadAll(r)
		require.NoError(t, err)

		require.Equal(t, msg, content)
	})

	t.Run("when the target do not exist", func(t *testing.T) {
		dir, err := ioutil.TempDir("", "configs")
		require.NoError(t, err)
		defer os.Remove(dir)

		target := filepath.Join(dir, "hello.txt")
		d := &DiskStore{target: target}

		msg := []byte("bonjour la famille")
		err = d.Save(bytes.NewReader(msg))
		require.NoError(t, err)

		content, err := ioutil.ReadFile(target)
		require.NoError(t, err)

		require.Equal(t, msg, content)
	})
}

func genFile(b []byte) (string, error) {
	dir, err := ioutil.TempDir("", "configs")
	if err != nil {
		return "", err
	}

	f, err := ioutil.TempFile(dir, "config-")
	if err != nil {
		return "", err
	}
	f.Write(b)
	name := f.Name()
	if err := f.Close(); err != nil {
		return "", err
	}

	return name, nil
}

func requireFilesCount(t *testing.T, dir string, l int) {
	files, err := ioutil.ReadDir(dir)
	require.NoError(t, err)
	require.Equal(t, l, len(files))
}
