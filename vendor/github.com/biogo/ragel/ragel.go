// Copyright ©2013 The bíogo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ragel provides helper functions and types for building ragel-based parsers.
//
// The ragel state machine compiler is available from http://www.complang.org/ragel/.
package ragel

import (
	"bufio"
	"errors"
	"io"
)

var (
	ErrNilData    = errors.New("ragel: nil value for expected ragel variable")
	ErrBufferFull = errors.New("ragel: buffer full")
	ErrNotFound   = errors.New("ragel: not found")
)

// BlockReader implements reading defined block size data into a provided slice and
// updating ragel state machine variables.
type BlockReader struct {
	r     io.Reader
	data  []byte
	buf   []byte
	p, pe *int
	eof   *int
}

// NewBlockReader returns a new BlockReader that reads from r. Pointers to the ragel
// state machine variables p, pe and eof, and a data slice must be provided. These
// variables are updated on each read. An error is returned if a pointer to any of the
// required ragel variables is nil.
func NewBlockReader(r io.Reader, p, pe, eof *int, data []byte) (*BlockReader, error) {
	if data == nil || p == nil || pe == nil || eof == nil {
		return nil, ErrNilData
	}
	*eof = -1
	return &BlockReader{
		r:    r,
		data: data,
		buf:  data[:0],
		p:    p,
		pe:   pe,
		eof:  eof,
	}, nil
}

// Buffered returns the number of bytes that can be processed in the current block.
func (r *BlockReader) Buffered() int { return len(r.buf) - *r.p }

// Read reads data from the underlying reader until the ragel data variable is full,
// first moving read but unprocessed data to the start of the data slice if the p ragel
// variable has passed len(data)/2. Read returns the number of bytes read and any
// error encountered during the read. ErrBufferFull is returned if the data variable
// is full prior to reading and after any potential data shift. Read sets the ragel eof
// variable to the value of pe if the read returns an io.EOF error.
func (r *BlockReader) Read() (n int, err error) {
	p, data := *r.p, r.data
	if p > len(data)/2 {
		copy(data, data[p:len(r.buf)])
		*r.p, *r.pe = 0, *r.pe-p
		r.buf = r.buf[0 : len(r.buf)-p]
	}
	if len(r.buf) == cap(r.buf) {
		return 0, ErrBufferFull
	}
	n, err = r.r.Read(data[len(r.buf):])
	r.buf = r.buf[:len(r.buf)+n]
	*r.pe = len(r.buf)
	if err == io.EOF {
		*r.eof = *r.pe
	}
	return n, err
}

// BackupTo scans backwards from the position pe in the data variable to the first
// instance of b, setting pe to the next position from the found byte. ErrNotFound
// is returned and pe is not updated if b is not found.
func (r *BlockReader) BackupTo(b byte) error {
	if len(r.data) == 0 {
		return ErrNotFound
	}
	i, p := *r.pe-1, *r.p
	for ; i >= p && r.data[i] != b; i-- {
	}
	if i == p-1 && r.data[p] != b {
		return ErrNotFound
	}
	*r.pe = i + 1
	return nil
}

// AppendReader implements reading arbitrarily sized, byte delimited data into a slice and
// updating ragel state machine variables.
type AppendReader struct {
	r     *bufio.Reader
	delim byte
	data  *[]byte
	p, pe *int
	eof   *int
}

// NewAppendReader returns a new AppenReader that reads from r. Pointers to the ragel
// state machine variables p, pe, eof and data slice must be provided. These variables
// are updated on each read. An error is returned if a pointer to any of the required
// ragel variables is nil.
func NewAppendReader(r *bufio.Reader, p, pe, eof *int, data *[]byte, delim byte) (*AppendReader, error) {
	if data == nil || p == nil || pe == nil || eof == nil {
		return nil, ErrNilData
	}
	*data, *eof = (*data)[:0], -1
	return &AppendReader{
		r:     r,
		delim: delim,
		data:  data,
		p:     p,
		pe:    pe,
		eof:   eof,
	}, nil
}

// Buffered returns the number of bytes that can be processed in the current block.
func (r *AppendReader) Buffered() int { return len(*r.data) - *r.p }

// Read reads data from the underlying reader until the the first instant of the
// AppendReader's delim byte growing the data slice as necessary, but first moving
// read but unprocessed data to the start of the data slice if the p ragel variable
// has passed len(data)/2. Read returns the number of bytes read and any error
// encountered during the read. io.ErrUnexpectedEOF is returned if data does not end
// in the specified delimeter. Read sets the ragel eof variable to the value of pe
// if the read returns an io.EOF or io.ErrUnexpectedEOF error.
func (r *AppendReader) Read() (n int, err error) {
	p, data := *r.p, *r.data
	if p > len(data)/2 {
		copy(data, data[p:len(data)])
		*r.p, *r.pe = 0, *r.pe-p
		data = data[0 : len(data)-p]
	}
	line, err := r.r.ReadBytes(r.delim)
	*r.data = append(data, line...)
	*r.pe = len(*r.data)
	if err == io.EOF {
		if len(line) != 0 {
			err = io.ErrUnexpectedEOF
		}
		*r.eof = *r.pe
	}
	return n, err
}

// BackupTo scans backwards from the position pe in the data variable to the first
// instance of b, setting pe to the next position from the found byte. ErrNotFound
// is returned and pe is not updated if b is not found.
func (r *AppendReader) BackupTo(b byte) error {
	data := *r.data
	if len(data) == 0 {
		return ErrNotFound
	}
	i, p := *r.pe-1, *r.p
	for ; i >= p && data[i] != b; i-- {
	}
	if i == p-1 && data[p] != b {
		return ErrNotFound
	}
	*r.pe = i + 1
	return nil
}

// BlockScanner implements reading defined block size data into a provided slice and
// updating ragel state machine variables.
type BlockScanner struct {
	r      io.Reader
	data   []byte
	buf    []byte
	p, pe  *int
	ts, te *int
	act    *int
	eof    *int
}

// NewBlockScanner returns a new BlockScanner that reads from r. Pointers to the ragel
// state machine variables p, pe, ts, te and eof, and a data slice must be provided. These
// variables are updated on each read. An error is returned if a pointer to any of the
// required ragel variables is nil.
func NewBlockScanner(r io.Reader, p, pe, ts, te, eof *int, data []byte) (*BlockScanner, error) {
	if data == nil || p == nil || pe == nil || ts == nil || te == nil || eof == nil {
		return nil, ErrNilData
	}
	*eof = -1
	return &BlockScanner{
		r:    r,
		data: data,
		buf:  data[:0],
		p:    p,
		pe:   pe,
		ts:   ts,
		te:   te,
		eof:  eof,
	}, nil
}

// Buffered returns the number of bytes that can be processed in the current block.
func (r *BlockScanner) Buffered() int { return len(r.buf) - *r.ts }

// Read reads data from the underlying reader until the ragel data variable is full,
// first moving read but unprocessed data to the start of the data array if the ts ragel
// variable is non-zero. Read returns the number of bytes read and any error encountered
// during the read. ErrBufferFull is returned if the data variable is full prior to
// reading and after any potential data shift. Read sets the ragel eof variable to the
// value of pe if the read returns an io.EOF error.
func (r *BlockScanner) Read() (n int, err error) {
	ts, data := *r.ts, r.data
	if ts != 0 {
		copy(data, data[ts:len(r.buf)])
		*r.p -= ts
		*r.pe -= ts
		*r.ts = 0
		*r.te -= ts
		r.buf = r.buf[0 : len(r.buf)-ts]
	}
	if len(r.buf) == cap(r.buf) {
		return 0, ErrBufferFull
	}
	n, err = r.r.Read(data[len(r.buf):])
	r.buf = r.buf[:len(r.buf)+n]
	*r.pe = len(r.buf)
	if err == io.EOF {
		*r.eof = *r.pe
	}
	return n, err
}

// AppendScanner implements reading arbitrarily sized, byte delimited data into a slice and
// updating ragel state machine variables.
type AppendScanner struct {
	r      *bufio.Reader
	delim  byte
	data   *[]byte
	p, pe  *int
	ts, te *int
	eof    *int
}

// NewAppendScanner returns a new AppenReader that reads from r. Pointers to the ragel
// state machine variables p, pe, ts, te, eof and data must be provided. These variables
// are updated on each read. An error is returned if a pointer to any of the required
// ragel variables is nil.
func NewAppendScanner(r *bufio.Reader, p, pe, ts, te, eof *int, data *[]byte, delim byte) (*AppendScanner, error) {
	if data == nil || p == nil || pe == nil || ts == nil || te == nil || eof == nil {
		return nil, ErrNilData
	}
	*data, *eof = (*data)[:0], -1
	return &AppendScanner{
		r:     r,
		delim: delim,
		data:  data,
		p:     p,
		pe:    pe,
		ts:    ts,
		te:    te,
		eof:   eof,
	}, nil
}

// Buffered returns the number of bytes that can be processed in the current block.
func (r *AppendScanner) Buffered() int { return len(*r.data) - *r.ts }

// Read reads data from the underlying reader until the the first instant of the
// AppendScanner's delim byte growing the data slice as necessary, but first moving
// read unprocessed data to the start of the data array if the ts ragel variable is
// non-zero. Read returns the number of bytes read and any error encountered during
// the read. io.ErrUnexpectedEOF is returned if data does not end in the specified
// delimeter. Read sets the ragel eof variable to the value of pe if the read returns
// an io.EOF or io.ErrUnexpectedEOF error.
func (r *AppendScanner) Read() (n int, err error) {
	ts, data := *r.ts, *r.data
	if ts != 0 {
		copy(data, data[ts:len(data)])
		*r.p -= ts
		*r.pe -= ts
		*r.ts = 0
		*r.te -= ts
		data = data[0 : len(data)-ts]
	}
	line, err := r.r.ReadBytes(r.delim)
	*r.data = append(data, line...)
	*r.pe = len(*r.data)
	if err == io.EOF {
		if len(line) != 0 {
			err = io.ErrUnexpectedEOF
		}
		*r.eof = *r.pe
	}
	return n, err
}
