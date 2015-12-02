// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package thrift

import (
	"fmt"
	"io"
)

type ProtocolError struct {
	Protocol string
	Message  string
}

func (e ProtocolError) Error() string {
	return fmt.Sprintf("thrift: [%s] %s", e.Protocol, e.Message)
}

type ProtocolBuilder interface {
	NewProtocolReader(io.Reader) ProtocolReader
	NewProtocolWriter(io.Writer) ProtocolWriter
}

type protocolBuilder struct {
	reader func(io.Reader) ProtocolReader
	writer func(io.Writer) ProtocolWriter
}

func NewProtocolBuilder(r func(io.Reader) ProtocolReader, w func(io.Writer) ProtocolWriter) ProtocolBuilder {
	return &protocolBuilder{
		reader: r,
		writer: w,
	}
}

func (p *protocolBuilder) NewProtocolReader(r io.Reader) ProtocolReader {
	return p.reader(r)
}

func (p *protocolBuilder) NewProtocolWriter(w io.Writer) ProtocolWriter {
	return p.writer(w)
}

type ProtocolReadWriter interface {
	ProtocolReader
	ProtocolWriter
}

type ProtocolWriter interface {
	WriteMessageBegin(name string, messageType byte, seqid int32) error
	WriteMessageEnd() error
	WriteStructBegin(name string) error
	WriteStructEnd() error
	WriteFieldBegin(name string, fieldType byte, id int16) error
	WriteFieldEnd() error
	WriteFieldStop() error
	WriteMapBegin(keyType byte, valueType byte, size int) error
	WriteMapEnd() error
	WriteListBegin(elementType byte, size int) error
	WriteListEnd() error
	WriteSetBegin(elementType byte, size int) error
	WriteSetEnd() error
	WriteBool(value bool) error
	WriteByte(value byte) error
	WriteI16(value int16) error
	WriteI32(value int32) error
	WriteI64(value int64) error
	WriteDouble(value float64) error
	WriteString(value string) error
	WriteBytes(value []byte) error
}

type ProtocolReader interface {
	ReadMessageBegin() (name string, messageType byte, seqid int32, err error)
	ReadMessageEnd() error
	ReadStructBegin() error
	ReadStructEnd() error
	ReadFieldBegin() (fieldType byte, id int16, err error)
	ReadFieldEnd() error
	ReadMapBegin() (keyType byte, valueType byte, size int, err error)
	ReadMapEnd() error
	ReadListBegin() (elementType byte, size int, err error)
	ReadListEnd() error
	ReadSetBegin() (elementType byte, size int, err error)
	ReadSetEnd() error
	ReadBool() (bool, error)
	ReadByte() (byte, error)
	ReadI16() (int16, error)
	ReadI32() (int32, error)
	ReadI64() (int64, error)
	ReadDouble() (float64, error)
	ReadString() (string, error)
	ReadBytes() ([]byte, error)
}
