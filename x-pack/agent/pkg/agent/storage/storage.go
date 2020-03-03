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
	"time"

	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/crypto"
)

const perms = 0600

type store interface {
	Save(io.Reader) error
}

type load interface {
	Load() (io.ReadCloser, error)
}

// NullStore this is only use to split the work into multiples PRs.
type NullStore struct{}

// Save takes the fleetConfig and persist it, will return an errors on failure.
func (m *NullStore) Save(_ io.Reader) error {
	return nil
}

type handlerFunc func(io.Reader) error

// HandlerStore take a function handler and wrap it into the store interface.
type HandlerStore struct {
	fn handlerFunc
}

// NewHandlerStore takes a function and wrap it into an handlerStore.
func NewHandlerStore(fn handlerFunc) *HandlerStore {
	return &HandlerStore{fn: fn}
}

// Save calls the handler.
func (h *HandlerStore) Save(in io.Reader) error {
	return h.fn(in)
}

// ReplaceOnSuccessStore takes a target file, a replacement content and a wrapped store. This
// store is useful if you want to trigger an action to replace another file when the wrapped store save method
// is successful. This store will take care of making a backup copy of the target file and will not
// override the content of the target if the target has already the same content. If an error happen,
// we will not replace the file.
type ReplaceOnSuccessStore struct {
	target      string
	replaceWith []byte

	wrapped store
}

// NewReplaceOnSuccessStore takes a target file and a replacement content and will replace the target
// file content if the wrapped store execution is done without any error.
func NewReplaceOnSuccessStore(target string, replaceWith []byte, wrapped store) *ReplaceOnSuccessStore {
	return &ReplaceOnSuccessStore{
		target:      target,
		replaceWith: replaceWith,
		wrapped:     wrapped,
	}
}

// Save will replace a target file with new content if the wrapped store is successful.
func (r *ReplaceOnSuccessStore) Save(in io.Reader) error {
	// get original permission
	s, err := os.Stat(r.target)

	// Ensure we can read the target files before delegating any call to the wrapped store.
	target, err := ioutil.ReadFile(r.target)
	if err != nil {
		return errors.New(err,
			fmt.Sprintf("fail to read content of %s", r.target),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, r.target))
	}

	err = r.wrapped.Save(in)
	if err != nil {
		return err
	}

	if bytes.Equal(target, r.replaceWith) {
		return nil
	}

	// Windows is tricky with the characters permitted for the path and filename, so we have
	// to remove any colon from the string. We are using nanosec precision here because of automated
	// tools.
	const fsSafeTs = "2006-01-02T15-04-05.9999"

	ts := time.Now()
	backFilename := r.target + "." + ts.Format(fsSafeTs) + ".bak"
	if err := file.SafeFileRotate(backFilename, r.target); err != nil {
		return errors.New(err,
			fmt.Sprintf("could not backup %s", r.target),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, r.target))
	}

	fd, err := os.OpenFile(r.target, os.O_CREATE|os.O_WRONLY, s.Mode())
	if err != nil {
		// Rollback on any errors to minimize non working state.
		if err := file.SafeFileRotate(r.target, backFilename); err != nil {
			return errors.New(err,
				fmt.Sprintf("could not rollback %s to %s", backFilename, r.target),
				errors.TypeFilesystem,
				errors.M(errors.MetaKeyPath, r.target),
				errors.M("backup_path", backFilename))
		}
	}

	if _, err := fd.Write(r.replaceWith); err != nil {
		if err := file.SafeFileRotate(r.target, backFilename); err != nil {
			return errors.New(err,
				fmt.Sprintf("could not rollback %s to %s", backFilename, r.target),
				errors.TypeFilesystem,
				errors.M(errors.MetaKeyPath, r.target),
				errors.M("backup_path", backFilename))
		}
	}

	return nil
}

// DiskStore takes a persistedConfig and save it to a temporary files and replace the target file.
type DiskStore struct {
	target string
}

// NewDiskStore creates an unencrypted disk store.
func NewDiskStore(target string) *DiskStore {
	return &DiskStore{target: target}
}

// Save accepts a persistedConfig and saved it to a target file, to do so we will
// make a temporary files if the write is successful we are replacing the target file with the
// original content.
func (d *DiskStore) Save(in io.Reader) error {
	tmpFile := d.target + ".tmp"

	fd, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perms)
	if err != nil {
		return errors.New(err,
			fmt.Sprintf("could not save to %s", tmpFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, tmpFile))
	}
	defer fd.Close()

	// Always clean up the temporary file and ignore errors.
	defer os.Remove(tmpFile)

	if _, err := io.Copy(fd, in); err != nil {
		return errors.New(err, "could not save content on disk",
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, tmpFile))
	}

	if err := file.SafeFileRotate(d.target, tmpFile); err != nil {
		return errors.New(err,
			fmt.Sprintf("could not replace target file %s", d.target),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, d.target))
	}

	return nil
}

// Load return a io.ReadCloser for the target file.
func (d *DiskStore) Load() (io.ReadCloser, error) {
	return os.OpenFile(d.target, os.O_RDONLY, perms)
}

// EncryptedDiskStore save the persisted configuration and encrypt the data on disk.
type EncryptedDiskStore struct {
	target   string
	password []byte
}

// NewEncryptedDiskStore creates an encrypted disk store.
func NewEncryptedDiskStore(target string, password []byte) *EncryptedDiskStore {
	return &EncryptedDiskStore{target: target, password: password}
}

// Save accepts a persistedConfig, encrypt it and saved it to a target file, to do so we will
// make a temporary files if the write is successful we are replacing the target file with the
// original content.
func (d *EncryptedDiskStore) Save(in io.Reader) error {
	const perms = 0600

	tmpFile := d.target + ".tmp"

	fd, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perms)
	if err != nil {
		return errors.New(err,
			fmt.Sprintf("could not save to %s", tmpFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, tmpFile))
	}
	defer fd.Close()

	// Always clean up the temporary file and ignore errors.
	defer os.Remove(tmpFile)

	w, err := crypto.NewWriterWithDefaults(fd, d.password)
	if err != nil {
		return errors.New(err, "could not encrypt the data to disk",
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, tmpFile))
	}

	if _, err := io.Copy(w, in); err != nil {
		return errors.New(err, "could not save content on disk",
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, tmpFile))
	}

	if err := file.SafeFileRotate(d.target, tmpFile); err != nil {
		return errors.New(err,
			fmt.Sprintf("could not replace target file %s", d.target),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, d.target))
	}

	return nil
}

// Load return a io.ReadCloser that will take care on unencrypting the data.
func (d *EncryptedDiskStore) Load() (io.ReadCloser, error) {
	fd, err := os.OpenFile(d.target, os.O_RDONLY|os.O_CREATE, perms)
	if err != nil {
		return nil, errors.New(err,
			fmt.Sprintf("could not open %s", d.target),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, d.target))
	}

	r, err := crypto.NewReaderWithDefaults(fd, d.password)
	if err != nil {
		fd.Close()
		return nil, errors.New(err,
			fmt.Sprintf("could not decode file %s", d.target),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, d.target))
	}

	return r, nil
}
