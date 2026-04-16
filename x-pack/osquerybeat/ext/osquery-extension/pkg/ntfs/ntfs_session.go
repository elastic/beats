// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package ntfs

import (
	"io"

	"www.velocidex.com/golang/go-ntfs/parser"
)

type NTFSSession struct {
	ctx    *parser.NTFSContext
	reader *windowsVolumeReader
}

func (s *NTFSSession) Close() {
	if s.reader != nil {
		s.reader.Close()
	}
}

// Context returns the underlying NTFSContext for direct access to MFT operations.
func (s *NTFSSession) Context() *parser.NTFSContext {
	return s.ctx
}

// RawReader returns the underlying raw volume reader, bypassing the paged cache layer.
// Use this for large sequential reads where the PagedReader's small page size would
// cause excessive numbers of individual I/O operations.
func (s *NTFSSession) RawReader() io.ReaderAt {
	return s.reader
}

func NewNTFSSession(driveLetter string) (*NTFSSession, error) {
	driveLetter, err := normalizeDriveLetter(driveLetter)
	if err != nil {
		return nil, err
	}

	reader, err := NewVolumeReader(driveLetter)
	if err != nil {
		return nil, err
	}

	// Close the reader if we fail to initialize the session
	initialized := false
	defer func() {
		if !initialized {
			reader.Close()
		}
	}()

	pagedReader, err := parser.NewPagedReader(reader, 1024, 10000)
	if err != nil {
		return nil, err
	}
	ctx, err := parser.GetNTFSContext(pagedReader, 0)
	if err != nil {
		return nil, err
	}

	initialized = true
	return &NTFSSession{ctx: ctx, reader: reader}, nil
}
