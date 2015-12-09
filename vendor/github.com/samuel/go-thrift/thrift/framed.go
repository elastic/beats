// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package thrift

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	// DefaultMaxFrameSize is the default max size for frames when using the FramedReadWriteCloser
	DefaultMaxFrameSize = 1024 * 1024
)

type ErrFrameTooBig struct {
	Size, MaxSize int64
}

func (e ErrFrameTooBig) Error() string {
	return fmt.Sprintf("thrift: frame size while reading over allowed size (%d > %d)", e.Size, e.MaxSize)
}

type Flusher interface {
	Flush() error
}

type FramedReadWriteCloser struct {
	wrapped       io.ReadWriteCloser
	limitedReader *io.LimitedReader
	maxFrameSize  int64
	rtmp          []byte
	wtmp          []byte
	rbuf          *bytes.Buffer
	wbuf          *bytes.Buffer
}

func NewFramedReadWriteCloser(wrapped io.ReadWriteCloser, maxFrameSize int) *FramedReadWriteCloser {
	if maxFrameSize == 0 {
		maxFrameSize = DefaultMaxFrameSize
	}
	return &FramedReadWriteCloser{
		wrapped:       wrapped,
		limitedReader: &io.LimitedReader{R: wrapped, N: 0},
		maxFrameSize:  int64(maxFrameSize),
		rtmp:          make([]byte, 4),
		wtmp:          make([]byte, 4),
		rbuf:          &bytes.Buffer{},
		wbuf:          &bytes.Buffer{},
	}
}

func (f *FramedReadWriteCloser) Read(p []byte) (int, error) {
	if err := f.fillBuffer(); err != nil {
		return 0, err
	}
	return f.rbuf.Read(p)
}

func (f *FramedReadWriteCloser) ReadByte() (byte, error) {
	if err := f.fillBuffer(); err != nil {
		return 0, err
	}
	return f.rbuf.ReadByte()
}

func (f *FramedReadWriteCloser) fillBuffer() error {
	if f.rbuf.Len() > 0 {
		return nil
	}

	f.rbuf.Reset()
	if _, err := io.ReadFull(f.wrapped, f.rtmp); err != nil {
		return err
	}
	frameSize := int64(binary.BigEndian.Uint32(f.rtmp))
	if frameSize > f.maxFrameSize {
		return ErrFrameTooBig{frameSize, f.maxFrameSize}
	}
	f.limitedReader.N = frameSize
	written, err := io.Copy(f.rbuf, f.limitedReader)
	if err != nil {
		return err
	}
	if written < frameSize {
		return io.EOF
	}
	return nil
}

func (f *FramedReadWriteCloser) Write(p []byte) (int, error) {
	n, err := f.wbuf.Write(p)
	if err != nil {
		return n, err
	}
	if ln := int64(f.wbuf.Len()); ln > f.maxFrameSize {
		return n, &ErrFrameTooBig{ln, f.maxFrameSize}
	}
	return n, nil
}

func (f *FramedReadWriteCloser) Close() error {
	return f.wrapped.Close()
}

func (f *FramedReadWriteCloser) Flush() error {
	frameSize := uint32(f.wbuf.Len())
	if frameSize > 0 {
		binary.BigEndian.PutUint32(f.wtmp, frameSize)
		if _, err := f.wrapped.Write(f.wtmp); err != nil {
			return err
		}
		_, err := io.Copy(f.wrapped, f.wbuf)
		f.wbuf.Reset()
		return err
	}
	return nil
}
