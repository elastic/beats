// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package thrift

import (
	"encoding/binary"
	"io"
	"math"
)

const (
	versionMask uint32 = 0xffff0000
	version1    uint32 = 0x80010000
	typeMask    uint32 = 0x000000ff
)

const (
	maxMessageNameSize = 128
)

type binaryProtocolWriter struct {
	w      io.Writer
	strict bool
	buf    []byte
}

type binaryProtocolReader struct {
	r      io.Reader
	strict bool
	buf    []byte
}

var BinaryProtocol = NewProtocolBuilder(
	func(r io.Reader) ProtocolReader { return NewBinaryProtocolReader(r, false) },
	func(w io.Writer) ProtocolWriter { return NewBinaryProtocolWriter(w, true) },
)

func NewBinaryProtocolWriter(w io.Writer, strict bool) ProtocolWriter {
	p := &binaryProtocolWriter{
		w:      w,
		strict: strict,
		buf:    make([]byte, 32),
	}
	return p
}

func NewBinaryProtocolReader(r io.Reader, strict bool) ProtocolReader {
	p := &binaryProtocolReader{
		r:      r,
		strict: strict,
		buf:    make([]byte, 32),
	}
	return p
}

func (p *binaryProtocolWriter) WriteMessageBegin(name string, messageType byte, seqid int32) error {
	if p.strict {
		if err := p.WriteI32(int32(version1 | uint32(messageType))); err != nil {
			return err
		}
		if err := p.WriteString(name); err != nil {
			return err
		}
	} else {
		if err := p.WriteString(name); err != nil {
			return err
		}
		if err := p.WriteByte(messageType); err != nil {
			return err
		}
	}
	return p.WriteI32(seqid)
}

func (p *binaryProtocolWriter) WriteMessageEnd() error {
	return nil
}

func (p *binaryProtocolWriter) WriteStructBegin(name string) error {
	return nil
}

func (p *binaryProtocolWriter) WriteStructEnd() error {
	return nil
}

func (p *binaryProtocolWriter) WriteFieldBegin(name string, fieldType byte, id int16) error {
	if err := p.WriteByte(fieldType); err != nil {
		return err
	}
	return p.WriteI16(id)
}

func (p *binaryProtocolWriter) WriteFieldEnd() error {
	return nil
}

func (p *binaryProtocolWriter) WriteFieldStop() error {
	return p.WriteByte(TypeStop)
}

func (p *binaryProtocolWriter) WriteMapBegin(keyType byte, valueType byte, size int) error {
	if err := p.WriteByte(keyType); err != nil {
		return err
	}
	if err := p.WriteByte(valueType); err != nil {
		return err
	}
	return p.WriteI32(int32(size))
}

func (p *binaryProtocolWriter) WriteMapEnd() error {
	return nil
}

func (p *binaryProtocolWriter) WriteListBegin(elementType byte, size int) error {
	if err := p.WriteByte(elementType); err != nil {
		return err
	}
	return p.WriteI32(int32(size))
}

func (p *binaryProtocolWriter) WriteListEnd() error {
	return nil
}

func (p *binaryProtocolWriter) WriteSetBegin(elementType byte, size int) error {
	if err := p.WriteByte(elementType); err != nil {
		return err
	}
	return p.WriteI32(int32(size))
}

func (p *binaryProtocolWriter) WriteSetEnd() error {
	return nil
}

func (p *binaryProtocolWriter) WriteBool(value bool) error {
	if value {
		return p.WriteByte(1)
	}
	return p.WriteByte(0)
}

func (p *binaryProtocolWriter) WriteByte(value byte) error {
	b := p.buf
	if b == nil {
		b = []byte{value}
	} else {
		b[0] = value
	}
	_, err := p.w.Write(b[:1])
	return err
}

func (p *binaryProtocolWriter) WriteI16(value int16) error {
	b := p.buf
	if b == nil {
		b = []byte{0, 0}
	}
	binary.BigEndian.PutUint16(b, uint16(value))
	_, err := p.w.Write(b[:2])
	return err
}

func (p *binaryProtocolWriter) WriteI32(value int32) error {
	b := p.buf
	if b == nil {
		b = []byte{0, 0, 0, 0}
	}
	binary.BigEndian.PutUint32(b, uint32(value))
	_, err := p.w.Write(b[:4])
	return err
}

func (p *binaryProtocolWriter) WriteI64(value int64) error {
	b := p.buf
	if b == nil {
		b = []byte{0, 0, 0, 0, 0, 0, 0, 0}
	}
	binary.BigEndian.PutUint64(b, uint64(value))
	_, err := p.w.Write(b[:8])
	return err
}

func (p *binaryProtocolWriter) WriteDouble(value float64) error {
	b := p.buf
	if b == nil {
		b = []byte{0, 0, 0, 0, 0, 0, 0, 0}
	}
	binary.BigEndian.PutUint64(b, math.Float64bits(value))
	_, err := p.w.Write(b[:8])
	return err
}

func (p *binaryProtocolWriter) WriteString(value string) error {
	if len(value) <= len(p.buf) {
		if err := p.WriteI32(int32(len(value))); err != nil {
			return err
		}
		n := copy(p.buf, value)
		_, err := p.w.Write(p.buf[:n])
		return err
	}
	return p.WriteBytes([]byte(value))
}

func (p *binaryProtocolWriter) WriteBytes(value []byte) error {
	if err := p.WriteI32(int32(len(value))); err != nil {
		return err
	}
	_, err := p.w.Write(value)
	return err
}

func (p *binaryProtocolReader) ReadMessageBegin() (name string, messageType byte, seqid int32, err error) {
	size, e := p.ReadI32()
	if e != nil {
		err = e
		return
	}
	if size < 0 {
		version := uint32(size) & versionMask
		if version != version1 {
			err = ProtocolError{"BinaryProtocol", "bad version in ReadMessageBegin"}
			return
		}
		messageType = byte(uint32(size) & typeMask)
		if name, err = p.ReadString(); err != nil {
			return
		}
	} else {
		if p.strict {
			err = ProtocolError{"BinaryProtocol", "no protocol version header"}
			return
		}
		if size > maxMessageNameSize {
			err = ProtocolError{"BinaryProtocol", "message name exceeds max size"}
			return
		}
		nameBytes := make([]byte, size)
		if _, err = p.r.Read(nameBytes); err != nil {
			return
		}
		name = string(nameBytes)
		if messageType, err = p.ReadByte(); err != nil {
			return
		}
	}
	seqid, err = p.ReadI32()
	return
}

func (p *binaryProtocolReader) ReadMessageEnd() error {
	return nil
}

func (p *binaryProtocolReader) ReadStructBegin() error {
	return nil
}

func (p *binaryProtocolReader) ReadStructEnd() error {
	return nil
}

func (p *binaryProtocolReader) ReadFieldBegin() (fieldType byte, id int16, err error) {
	if fieldType, err = p.ReadByte(); err != nil || fieldType == TypeStop {
		return
	}
	id, err = p.ReadI16()
	return
}

func (p *binaryProtocolReader) ReadFieldEnd() error {
	return nil
}

func (p *binaryProtocolReader) ReadMapBegin() (keyType byte, valueType byte, size int, err error) {
	if keyType, err = p.ReadByte(); err != nil {
		return
	}
	if valueType, err = p.ReadByte(); err != nil {
		return
	}
	var sz int32
	sz, err = p.ReadI32()
	size = int(sz)
	return
}

func (p *binaryProtocolReader) ReadMapEnd() error {
	return nil
}

func (p *binaryProtocolReader) ReadListBegin() (elementType byte, size int, err error) {
	if elementType, err = p.ReadByte(); err != nil {
		return
	}
	var sz int32
	sz, err = p.ReadI32()
	size = int(sz)
	return
}

func (p *binaryProtocolReader) ReadListEnd() error {
	return nil
}

func (p *binaryProtocolReader) ReadSetBegin() (elementType byte, size int, err error) {
	if elementType, err = p.ReadByte(); err != nil {
		return
	}
	var sz int32
	sz, err = p.ReadI32()
	size = int(sz)
	return
}

func (p *binaryProtocolReader) ReadSetEnd() error {
	return nil
}

func (p *binaryProtocolReader) ReadBool() (bool, error) {
	if b, e := p.ReadByte(); e != nil {
		return false, e
	} else if b != 0 {
		return true, nil
	}
	return false, nil
}

func (p *binaryProtocolReader) ReadByte() (value byte, err error) {
	_, err = io.ReadFull(p.r, p.buf[:1])
	value = p.buf[0]
	return
}

func (p *binaryProtocolReader) ReadI16() (value int16, err error) {
	_, err = io.ReadFull(p.r, p.buf[:2])
	value = int16(binary.BigEndian.Uint16(p.buf))
	return
}

func (p *binaryProtocolReader) ReadI32() (value int32, err error) {
	_, err = io.ReadFull(p.r, p.buf[:4])
	value = int32(binary.BigEndian.Uint32(p.buf))
	return
}

func (p *binaryProtocolReader) ReadI64() (value int64, err error) {
	_, err = io.ReadFull(p.r, p.buf[:8])
	value = int64(binary.BigEndian.Uint64(p.buf))
	return
}

func (p *binaryProtocolReader) ReadDouble() (value float64, err error) {
	_, err = io.ReadFull(p.r, p.buf[:8])
	value = math.Float64frombits(binary.BigEndian.Uint64(p.buf))
	return
}

func (p *binaryProtocolReader) ReadString() (string, error) {
	ln, err := p.ReadI32()
	if err != nil || ln == 0 {
		return "", err
	}
	if ln < 0 {
		return "", ProtocolError{"BinaryProtocol", "negative length while reading string"}
	}
	b := p.buf
	if int(ln) > len(b) {
		b = make([]byte, ln)
	} else {
		b = b[:ln]
	}
	if _, err := io.ReadFull(p.r, b); err != nil {
		return "", err
	}
	return string(b), nil
}

func (p *binaryProtocolReader) ReadBytes() ([]byte, error) {
	ln, err := p.ReadI32()
	if err != nil || ln == 0 {
		return nil, err
	}
	if ln < 0 {
		return nil, ProtocolError{"BinaryProtocol", "negative length while reading bytes"}
	}
	b := make([]byte, ln)
	if _, err := io.ReadFull(p.r, b); err != nil {
		return nil, err
	}
	return b, nil
}
