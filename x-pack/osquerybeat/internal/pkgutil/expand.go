// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pkgutil

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/cpio"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/xar"
)

var ErrPayloadNotFound = errors.New("pkg Payload file not found")

func Expand(pkgFile, dstDir string) error {
	f, err := os.Open(pkgFile)
	if err != nil {
		return err
	}
	defer f.Close()

	xr, err := xar.NewReader(f)

	if err != nil {
		return err
	}

	var payloadFile *xar.File
	for i := 0; i < len(xr.Files); i++ {
		f := xr.Files[i]
		if f.Name == "Payload" {
			payloadFile = &f
			break
		}
	}

	if payloadFile == nil {
		return ErrPayloadNotFound
	}

	return expandPayload(payloadFile, dstDir)
}

func expandPayload(f *xar.File, dstDir string) error {
	gzr, err := gzip.NewReader(f.Body)
	if err != nil {
		return err
	}

	err = os.MkdirAll(dstDir, 0755)
	if err != nil {
		return err
	}

	r := cpio.NewReader(gzr)
	var entry cpio.Entry
	for entry, err = r.Next(); err == nil; entry, err = r.Next() {
		if entry.FilePath == "." {
			continue
		}

		// Reject entries whose path escapes the destination directory to
		// guard against path traversal (CWE-22).
		if !filepath.IsLocal(entry.FilePath) {
			return fmt.Errorf("illegal file path in pkg payload: %v", entry.FilePath)
		}
		fp := filepath.Join(dstDir, entry.FilePath)

		body := entry.Body
		// Ignore symlinks, not needed for our purposes
		if entry.FileMode.IsDir() {
			err = os.MkdirAll(fp, entry.FileMode.Perm())
			if err != nil {
				return err
			}
		} else if entry.FileMode.IsRegular() {
			body = nil
			err := writeFile(fp, &entry)
			if err != nil {
				return err
			}
		}

		if body != nil {
			// Discarding the body, need to read in full
			_, _ = io.Copy(io.Discard, entry.Body)
		}
	}

	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	return nil
}

func writeFile(fp string, entry *cpio.Entry) error {
	err := writeFileContent(fp, entry.Body, entry.FileMode.Perm())
	if err != nil {
		return err
	}

	return os.Chtimes(fp, time.Now().Local(), entry.FileMtime)
}

func writeFileContent(fp string, r io.Reader, mode fs.FileMode) error {
	f, err := os.OpenFile(fp, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	if err != nil {
		return err
	}
	return nil
}
