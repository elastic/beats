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

	"github.com/hectane/go-acl"

	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
)

// ReplaceOnSuccessStore takes a target file, a replacement content and a wrapped store. This
// store is useful if you want to trigger an action to replace another file when the wrapped store save method
// is successful. This store will take care of making a backup copy of the target file and will not
// override the content of the target if the target has already the same content. If an error happen,
// we will not replace the file.
type ReplaceOnSuccessStore struct {
	target      string
	replaceWith []byte

	wrapped Store
}

// NewReplaceOnSuccessStore takes a target file and a replacement content and will replace the target
// file content if the wrapped store execution is done without any error.
func NewReplaceOnSuccessStore(target string, replaceWith []byte, wrapped Store) *ReplaceOnSuccessStore {
	return &ReplaceOnSuccessStore{
		target:      target,
		replaceWith: replaceWith,
		wrapped:     wrapped,
	}
}

// Save will replace a target file with new content if the wrapped store is successful.
func (r *ReplaceOnSuccessStore) Save(in io.Reader) error {
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

	fd, err := os.OpenFile(r.target, os.O_CREATE|os.O_WRONLY, perms)
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
	if err := fd.Close(); err != nil {
		return errors.New(err, fmt.Sprintf("could not close target file after checking for access %s", r.target),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, r.target))
	}

	// create a temporary file with the new changes, if successful, will replace the target file.
	tmpFile := r.target + ".tmp"

	// Always clean up the temporary file and ignore errors.
	defer os.Remove(tmpFile)

	fdt, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perms)
	if err != nil {
		return errors.New(err,
			fmt.Sprintf("could not save to %s", tmpFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, tmpFile))
	}

	if _, err := fdt.Write(r.replaceWith); err != nil {
		if err := fdt.Close(); err != nil {
			return errors.New(err, fmt.Sprintf("could not close temporary file %s", tmpFile),
				errors.TypeFilesystem,
				errors.M(errors.MetaKeyPath, tmpFile))
		}
		return errors.New(err, fmt.Sprintf("could not succefully write new changes in temporary file %s", tmpFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, tmpFile))
	}

	if err := fdt.Sync(); err != nil {
		return errors.New(err,
			fmt.Sprintf("could not sync temporary file %s", tmpFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, tmpFile))
	}

	if err := fdt.Close(); err != nil {
		return errors.New(err, fmt.Sprintf("could not close temporary file %s", tmpFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, tmpFile))
	}

	if err := file.SafeFileRotate(r.target, tmpFile); err != nil {
		return errors.New(err,
			fmt.Sprintf("could not replace target file %s with %s", r.target, tmpFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, r.target),
			errors.M("backup_path", backFilename))
	}

	if err := acl.Chmod(r.target, perms); err != nil {
		return errors.New(err,
			fmt.Sprintf("could not set permissions target file %s", r.target),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, r.target))
	}

	return nil
}
