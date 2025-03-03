// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cpio

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"strconv"
	"time"
)

var magicHeader = []byte("070707")

var ErrInvalidData = errors.New("invalid cpio data")
var ErrInvalidHeader = errors.New("invalid cpio header")

type Reader struct {
	r io.Reader
}

type Entry struct {
	Magic     [6]byte
	Dev       [6]byte
	Ino       [6]byte
	Mode      [6]byte
	Uid       [6]byte
	Gid       [6]byte
	Nlink     [6]byte
	Rdev      [6]byte
	Mtime     [11]byte
	Namesize  [6]byte
	Filesize  [11]byte
	FileMode  fs.FileMode
	FileMtime time.Time
	FilePath  string
	Body      io.Reader // Body needs to be read fully before proceeding to the next element
}

func NewReader(r io.Reader) *Reader {
	return &Reader{
		r: r,
	}
}

func (r *Reader) Next() (e Entry, err error) {
	// Read header
	err = r.read(e.Magic[:])
	if err != nil {
		return e, err
	}

	if !bytes.Equal(e.Magic[:], magicHeader) {
		return e, ErrInvalidHeader
	}

	err = r.read(e.Dev[:])
	if err != nil {
		return e, err
	}

	err = r.read(e.Ino[:])
	if err != nil {
		return e, err
	}

	err = r.read(e.Mode[:])
	if err != nil {
		return e, err
	}
	e.FileMode, err = parseFileMode(e.Mode)
	if err != nil {
		return e, err
	}

	err = r.read(e.Uid[:])
	if err != nil {
		return e, err
	}

	err = r.read(e.Gid[:])
	if err != nil {
		return e, err
	}

	err = r.read(e.Nlink[:])
	if err != nil {
		return e, err
	}

	err = r.read(e.Rdev[:])
	if err != nil {
		return e, err
	}

	err = r.read(e.Mtime[:])
	if err != nil {
		return e, err
	}

	mtime, err := strconv.ParseInt(string(e.Mtime[:]), 8, 64)
	if err != nil {
		return e, err
	}

	e.FileMtime = time.Unix(mtime, 0)

	err = r.read(e.Namesize[:])
	if err != nil {
		return e, err
	}

	err = r.read(e.Filesize[:])
	if err != nil {
		return e, err
	}

	nameSize, err := strconv.ParseUint(string(e.Namesize[:]), 8, 64)
	if err != nil {
		return e, err
	}
	if nameSize == 0 {
		return e, ErrInvalidData
	}

	filePath := make([]byte, nameSize)

	err = r.read(filePath)
	if err != nil {
		return e, err
	}

	// Check filePath ends with '\x00'
	if filePath[len(filePath)-1] != '\x00' {
		return e, ErrInvalidData
	}

	e.FilePath = string(filePath[:len(filePath)-1])
	if e.FilePath == "TRAILER!!!" {
		return e, io.EOF
	}

	fileSize, err := strconv.ParseInt(string(e.Filesize[:]), 8, 64)
	if err != nil {
		return e, err
	}

	if fileSize != 0 {
		e.Body = io.LimitReader(r.r, fileSize)
	}
	return e, nil
}

func (r *Reader) read(b []byte) error {
	_, err := io.ReadAtLeast(r.r, b, len(b))
	return err
}
