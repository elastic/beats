// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package thrift

import (
	"errors"
	"fmt"
	"io"
)

var (
	ErrUnimplemented = errors.New("thrift: unimplemented")
)

type textProtocolWriter struct {
	w           io.Writer
	indentation string
}

func NewTextProtocolWriter(w io.Writer) ProtocolWriter {
	return &textProtocolWriter{w: w}
}

func (p *textProtocolWriter) indent() {
	p.indentation += "\t"
}

func (p *textProtocolWriter) unindent() {
	p.indentation = p.indentation[:len(p.indentation)-1]
}

func (p *textProtocolWriter) WriteMessageBegin(name string, messageType byte, seqid int32) error {
	fmt.Fprintf(p.w, "%sMessageBegin(%s, %d, %.8x)\n", p.indentation, name, messageType, seqid)
	p.indent()
	return nil
}

func (p *textProtocolWriter) WriteMessageEnd() error {
	p.unindent()
	fmt.Fprintf(p.w, "%sMessageEnd()\n", p.indentation)
	return nil
}

func (p *textProtocolWriter) WriteStructBegin(name string) error {
	fmt.Fprintf(p.w, "%sStructBegin(%s)\n", p.indentation, name)
	p.indent()
	return nil
}

func (p *textProtocolWriter) WriteStructEnd() error {
	p.unindent()
	fmt.Fprintf(p.w, "%sStructEnd()\n", p.indentation)
	return nil
}

func (p *textProtocolWriter) WriteFieldBegin(name string, fieldType byte, id int16) error {
	fmt.Fprintf(p.w, "%sFieldBegin(%s, %d, %d)\n", p.indentation, name, fieldType, id)
	p.indent()
	return nil
}

func (p *textProtocolWriter) WriteFieldEnd() error {
	p.unindent()
	fmt.Fprintf(p.w, "%sFieldEnd()\n", p.indentation)
	return nil
}

func (p *textProtocolWriter) WriteFieldStop() error {
	fmt.Fprintf(p.w, "%sFieldStop()\n", p.indentation)
	return nil
}

func (p *textProtocolWriter) WriteMapBegin(keyType byte, valueType byte, size int) error {
	fmt.Fprintf(p.w, "%sMapBegin(%d, %d, %d)\n", p.indentation, keyType, valueType, size)
	p.indent()
	return nil
}

func (p *textProtocolWriter) WriteMapEnd() error {
	p.unindent()
	fmt.Fprintf(p.w, "%sMapEnd()\n", p.indentation)
	return nil
}

func (p *textProtocolWriter) WriteListBegin(elementType byte, size int) error {
	fmt.Fprintf(p.w, "%sListBegin(%d, %d)\n", p.indentation, elementType, size)
	p.indent()
	return nil
}

func (p *textProtocolWriter) WriteListEnd() error {
	p.unindent()
	fmt.Fprintf(p.w, "%sListEnd()\n", p.indentation)
	return nil
}

func (p *textProtocolWriter) WriteSetBegin(elementType byte, size int) error {
	fmt.Fprintf(p.w, "%sSetBegin(%d, %d)\n", p.indentation, elementType, size)
	p.indent()
	return nil
}

func (p *textProtocolWriter) WriteSetEnd() error {
	p.unindent()
	fmt.Fprintf(p.w, "%sSetEnd()\n", p.indentation)
	return nil
}

func (p *textProtocolWriter) WriteBool(value bool) error {
	fmt.Fprintf(p.w, "%sBool(%+v)\n", p.indentation, value)
	return nil
}

func (p *textProtocolWriter) WriteByte(value byte) error {
	fmt.Fprintf(p.w, "%sByte(%d)\n", p.indentation, value)
	return nil
}

func (p *textProtocolWriter) WriteI16(value int16) error {
	fmt.Fprintf(p.w, "%sI16(%d)\n", p.indentation, value)
	return nil
}

func (p *textProtocolWriter) WriteI32(value int32) error {
	fmt.Fprintf(p.w, "%sI32(%d)\n", p.indentation, value)
	return nil
}

func (p *textProtocolWriter) WriteI64(value int64) error {
	fmt.Fprintf(p.w, "%sI64(%d)\n", p.indentation, value)
	return nil
}

func (p *textProtocolWriter) WriteDouble(value float64) error {
	fmt.Fprintf(p.w, "%sDouble(%f)\n", p.indentation, value)
	return nil
}

func (p *textProtocolWriter) WriteString(value string) error {
	fmt.Fprintf(p.w, "%sString(%s)\n", p.indentation, value)
	return nil
}

func (p *textProtocolWriter) WriteBytes(value []byte) error {
	fmt.Fprintf(p.w, "%sBytes(%+v)\n", p.indentation, value)
	return nil
}

func (p *textProtocolWriter) ReadMessageBegin() (name string, messageType byte, seqid int32, err error) {
	return "", 0, 0, ErrUnimplemented
}

func (p *textProtocolWriter) ReadMessageEnd() error {
	return ErrUnimplemented
}

func (p *textProtocolWriter) ReadStructBegin() error {
	return ErrUnimplemented
}

func (p *textProtocolWriter) ReadStructEnd() error {
	return ErrUnimplemented
}

func (p *textProtocolWriter) ReadFieldBegin() (fieldType byte, id int16, err error) {
	return 0, 0, ErrUnimplemented
}

func (p *textProtocolWriter) ReadFieldEnd() error {
	return ErrUnimplemented
}

func (p *textProtocolWriter) ReadMapBegin() (keyType byte, valueType byte, size int, err error) {
	return 0, 0, 0, ErrUnimplemented
}

func (p *textProtocolWriter) ReadMapEnd() error {
	return ErrUnimplemented
}

func (p *textProtocolWriter) ReadListBegin() (elementType byte, size int, err error) {
	return 0, 0, ErrUnimplemented
}

func (p *textProtocolWriter) ReadListEnd() error {
	return ErrUnimplemented
}

func (p *textProtocolWriter) ReadSetBegin() (elementType byte, size int, err error) {
	return 0, 0, ErrUnimplemented
}

func (p *textProtocolWriter) ReadSetEnd() error {
	return ErrUnimplemented
}

func (p *textProtocolWriter) ReadBool() (bool, error) {
	return false, ErrUnimplemented
}

func (p *textProtocolWriter) ReadByte() (byte, error) {
	return 0, ErrUnimplemented
}

func (p *textProtocolWriter) ReadI16() (int16, error) {
	return 0, ErrUnimplemented
}

func (p *textProtocolWriter) ReadI32() (int32, error) {
	return 0, ErrUnimplemented
}

func (p *textProtocolWriter) ReadI64() (int64, error) {
	return 0, ErrUnimplemented
}

func (p *textProtocolWriter) ReadDouble() (float64, error) {
	return 0.0, ErrUnimplemented
}

func (p *textProtocolWriter) ReadString() (string, error) {
	return "", ErrUnimplemented
}

func (p *textProtocolWriter) ReadBytes() ([]byte, error) {
	return nil, ErrUnimplemented
}
