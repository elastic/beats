// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package xar

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"io"
)

// Limited implementation of XAR, only need to be able to unpack the Payload part.
// There is no XAR signatures validation
// no recursive TOC traversal.
// Just need the top level files, effectively only one Payload file

var ErrInvalidHeader = errors.New("invalid xar header")

var magic = []byte("xar!")

const headerSize = 28

// https://en.wikipedia.org/wiki/Xar_(archiver)
type xarHeader struct {
	headerSize         uint16
	formatVersion      uint16
	tocCompressedLen   uint64
	tocUncompressedLen uint64
	checksumAlgo       uint32
}

type Reader struct {
	r     io.ReaderAt
	Files []File
}

type File struct {
	Name     string
	Encoding string
	Body     io.Reader
}

func NewReader(r io.ReaderAt) (*Reader, error) {
	xr := &Reader{
		r: r,
	}

	err := xr.readTOC()
	if err != nil {
		return nil, err
	}

	return xr, nil
}

func (r *Reader) readTOC() error {
	var (
		header [headerSize]byte
		xh     xarHeader
		x      xar
	)
	_, err := r.r.ReadAt(header[:], 0)
	if err != nil {
		return err
	}

	if !bytes.Equal(header[:4], magic) {
		return ErrInvalidHeader
	}

	xh.headerSize = binary.BigEndian.Uint16(header[4:6])
	xh.formatVersion = binary.BigEndian.Uint16(header[6:8])
	xh.tocCompressedLen = binary.BigEndian.Uint64(header[8:16])
	xh.tocUncompressedLen = binary.BigEndian.Uint64(header[16:24])
	xh.checksumAlgo = binary.BigEndian.Uint32(header[24:])

	tocCompressed := make([]byte, xh.tocCompressedLen)
	_, err = r.r.ReadAt(tocCompressed, int64(xh.headerSize))
	if err != nil {
		return err
	}

	zr, err := zlib.NewReader(bytes.NewBuffer(tocCompressed))
	if err != nil {
		return err
	}
	defer zr.Close()

	xdec := xml.NewDecoder(zr)

	err = xdec.Decode(&x)
	if err != nil {
		return err
	}

	heapStartOffset := uint64(xh.headerSize) + xh.tocCompressedLen

	// The reader is at the beginning of the heap section here
	var files []File
	if len(x.Toc.Files) > 0 {
		files = make([]File, 0, len(x.Toc.Files))
	}
	for _, tf := range x.Toc.Files {
		files = append(files, File{
			Name:     tf.Name,
			Encoding: tf.Data.Encoding.Style,
			Body:     io.NewSectionReader(r.r, int64(heapStartOffset+tf.Data.Offset), int64(tf.Data.Size)),
		})
	}
	r.Files = files

	return nil
}
