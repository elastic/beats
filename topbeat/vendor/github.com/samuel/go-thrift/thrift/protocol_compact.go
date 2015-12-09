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
	compactProtocolID      = 0x82
	compactVersion         = 1
	compactVersionMask     = 0x1f
	compactTypeMask        = 0xe0
	compactTypeShiftAmount = 5
)

const (
	ctStop = iota
	ctTrue
	ctFalse
	ctByte
	ctI16
	ctI32
	ctI64
	ctDouble
	ctBinary
	ctList
	ctSet
	ctMap
	ctStruct
)

var thriftTypeToCompactType []byte
var compactTypeToThriftType []byte

func init() {
	thriftTypeToCompactType = make([]byte, 16)
	thriftTypeToCompactType[TypeStop] = ctStop
	thriftTypeToCompactType[TypeBool] = ctTrue
	thriftTypeToCompactType[TypeByte] = ctByte
	thriftTypeToCompactType[TypeI16] = ctI16
	thriftTypeToCompactType[TypeI32] = ctI32
	thriftTypeToCompactType[TypeI64] = ctI64
	thriftTypeToCompactType[TypeDouble] = ctDouble
	thriftTypeToCompactType[TypeString] = ctBinary
	thriftTypeToCompactType[TypeStruct] = ctStruct
	thriftTypeToCompactType[TypeList] = ctList
	thriftTypeToCompactType[TypeSet] = ctSet
	thriftTypeToCompactType[TypeMap] = ctMap
	compactTypeToThriftType = make([]byte, 16)
	compactTypeToThriftType[ctStop] = TypeStop
	compactTypeToThriftType[ctTrue] = TypeBool
	compactTypeToThriftType[ctFalse] = TypeBool
	compactTypeToThriftType[ctByte] = TypeByte
	compactTypeToThriftType[ctI16] = TypeI16
	compactTypeToThriftType[ctI32] = TypeI32
	compactTypeToThriftType[ctI64] = TypeI64
	compactTypeToThriftType[ctDouble] = TypeDouble
	compactTypeToThriftType[ctBinary] = TypeString
	compactTypeToThriftType[ctStruct] = TypeStruct
	compactTypeToThriftType[ctList] = TypeList
	compactTypeToThriftType[ctSet] = TypeSet
	compactTypeToThriftType[ctMap] = TypeMap
}

type compactProtocolWriter struct {
	w           io.Writer
	lastFieldID int16
	boolFid     int16
	boolValue   bool
	structs     []int16
	container   []int
	buf         []byte
}

type compactProtocolReader struct {
	r           io.Reader
	lastFieldID int16
	boolFid     int16
	boolValue   bool
	structs     []int16
	container   []int
	buf         []byte
}

var CompactProtocol = NewProtocolBuilder(NewCompactProtocolReader, NewCompactProtocolWriter)

func NewCompactProtocolWriter(w io.Writer) ProtocolWriter {
	return &compactProtocolWriter{
		w:           w,
		lastFieldID: 0,
		boolFid:     -1,
		boolValue:   false,
		structs:     make([]int16, 0, 8),
		container:   make([]int, 0, 8),
		buf:         make([]byte, 64),
	}
}

func NewCompactProtocolReader(r io.Reader) ProtocolReader {
	return &compactProtocolReader{
		r:           r,
		lastFieldID: 0,
		boolFid:     -1,
		boolValue:   false,
		structs:     make([]int16, 0, 8),
		container:   make([]int, 0, 8),
		buf:         make([]byte, 64),
	}
}

func (p *compactProtocolWriter) writeVarint(value int64) (err error) {
	n := binary.PutVarint(p.buf, value)
	_, err = p.w.Write(p.buf[:n])
	return
}

func (p *compactProtocolWriter) writeUvarint(value uint64) (err error) {
	n := binary.PutUvarint(p.buf, value)
	_, err = p.w.Write(p.buf[:n])
	return
}

// Write a message header to the wire. Compact Protocol messages contain the
// protocol version so we can migrate forwards in the future if need be.
func (p *compactProtocolWriter) WriteMessageBegin(name string, messageType byte, seqid int32) (err error) {
	if err = p.writeByteDirect(compactProtocolID); err != nil {
		return
	}
	if err = p.writeByteDirect(compactVersion | (messageType << compactTypeShiftAmount)); err != nil {
		return
	}
	if err = p.writeUvarint(uint64(seqid)); err != nil {
		return
	}
	err = p.WriteString(name)
	return
}

// Write a struct begin. This doesn't actually put anything on the wire. We
// use it as an opportunity to put special placeholder markers on the field
// stack so we can get the field id deltas correct.
func (p *compactProtocolWriter) WriteStructBegin(name string) error {
	p.structs = append(p.structs, p.lastFieldID)
	p.lastFieldID = 0
	return nil
}

// Write a struct end. This doesn't actually put anything on the wire. We use
// this as an opportunity to pop the last field from the current struct off
// of the field stack.
func (p *compactProtocolWriter) WriteStructEnd() error {
	if len(p.structs) == 0 {
		return ProtocolError{"CompactProtocol", "Struct end without matching begin"}
	}
	fid := p.structs[len(p.structs)-1]
	p.structs = p.structs[:len(p.structs)-1]
	p.lastFieldID = fid
	return nil
}

// Write a field header containing the field id and field type. If the
// difference between the current field id and the last one is small (< 15),
// then the field id will be encoded in the 4 MSB as a delta. Otherwise, the
// field id will follow the type header as a zigzag varint.
func (p *compactProtocolWriter) WriteFieldBegin(name string, fieldType byte, id int16) error {
	if fieldType == TypeBool {
		// we want to possibly include the value, so we'll wait.
		p.boolFid = id
		return nil
	}
	return p.writeFieldBeginInternal(name, fieldType, id, 0xff)
}

// The workhorse of writeFieldBegin. It has the option of doing a
// 'type override' of the type header. This is used specifically in the
// boolean field case.
func (p *compactProtocolWriter) writeFieldBeginInternal(name string, fieldType byte, id int16, typeOverride byte) error {
	// if there's a type override, use that.
	typeToWrite := typeOverride
	if typeToWrite == 0xff {
		typeToWrite = thriftTypeToCompactType[fieldType]
	}

	// check if we can use delta encoding for the field id
	if id > p.lastFieldID && id-p.lastFieldID <= 15 {
		// write them together
		if err := p.writeByteDirect(byte((id-p.lastFieldID)<<4 | int16(typeToWrite))); err != nil {
			return err
		}
	} else {
		// write them separate
		if err := p.writeByteDirect(byte(typeToWrite)); err != nil {
			return err
		}
		if err := p.WriteI16(id); err != nil {
			return err
		}
	}

	p.lastFieldID = id
	return nil
}

// Write the STOP symbol so we know there are no more fields in this struct.
func (p *compactProtocolWriter) WriteFieldStop() error {
	return p.writeByteDirect(TypeStop)
}

// Write a map header. If the map is empty, omit the key and value type
// headers, as we don't need any additional information to skip it.
func (p *compactProtocolWriter) WriteMapBegin(keyType byte, valueType byte, size int) error {
	if size == 0 {
		return p.writeByteDirect(0)
	}
	if err := p.writeUvarint(uint64(size)); err != nil {
		return err
	}
	return p.writeByteDirect(byte(thriftTypeToCompactType[keyType]<<4 | thriftTypeToCompactType[valueType]))
}

// Write a list header.
func (p *compactProtocolWriter) WriteListBegin(elementType byte, size int) error {
	return p.writeCollectionBegin(elementType, size)
}

// Write a set header.
func (p *compactProtocolWriter) WriteSetBegin(elementType byte, size int) error {
	return p.writeCollectionBegin(elementType, size)
}

// Write a boolean value. Potentially, this could be a boolean field, in
// which case the field header info isn't written yet. If so, decide what the
// right type header is for the value and then write the field header.
// Otherwise, write a single byte.
func (p *compactProtocolWriter) WriteBool(value bool) error {
	fieldType := byte(ctFalse)
	if value {
		fieldType = ctTrue
	}
	if p.boolFid >= 0 {
		// we haven't written the field header yet
		return p.writeFieldBeginInternal("bool", TypeBool, p.boolFid, fieldType)
	}
	return p.writeByteDirect(fieldType)
}

func (p *compactProtocolWriter) WriteByte(value byte) error {
	return p.writeByteDirect(value)
}

func (p *compactProtocolWriter) WriteI16(value int16) error {
	return p.writeVarint(int64(value))
}

func (p *compactProtocolWriter) WriteI32(value int32) error {
	return p.writeVarint(int64(value))
}

func (p *compactProtocolWriter) WriteI64(value int64) error {
	return p.writeVarint(value)
}

func (p *compactProtocolWriter) WriteDouble(value float64) (err error) {
	b := p.buf
	binary.BigEndian.PutUint64(b, math.Float64bits(value))
	_, err = p.w.Write(b[:8])
	return
}

// Write a string to the wire with a varint size preceeding.
func (p *compactProtocolWriter) WriteString(value string) error {
	return p.WriteBytes([]byte(value))
}

// Write a byte array, using a varint for the size.
func (p *compactProtocolWriter) WriteBytes(value []byte) error {
	if err := p.writeUvarint(uint64(len(value))); err != nil {
		return err
	}
	_, err := p.w.Write(value)
	return err
}

func (p *compactProtocolWriter) WriteMessageEnd() error {
	return nil
}

func (p *compactProtocolWriter) WriteMapEnd() error {
	return nil
}

func (p *compactProtocolWriter) WriteListEnd() error {
	return nil
}

func (p *compactProtocolWriter) WriteSetEnd() error {
	return nil
}

func (p *compactProtocolWriter) WriteFieldEnd() error {
	return nil
}

// Abstract method for writing the start of lists and sets. List and sets on
// the wire differ only by the type indicator.
func (p *compactProtocolWriter) writeCollectionBegin(elemType byte, size int) error {
	if size <= 14 {
		return p.writeByteDirect(byte(size)<<4 | thriftTypeToCompactType[elemType])
	}
	if err := p.writeByteDirect(0xf0 | thriftTypeToCompactType[elemType]); err != nil {
		return err
	}
	return p.writeUvarint(uint64(size))
}

// Writes a byte without any possiblity of all that field header nonsense.
// Used internally by other writing methods that know they need to write a byte.
func (p *compactProtocolWriter) writeByteDirect(value byte) error {
	p.buf[0] = value
	_, err := p.w.Write(p.buf[:1])
	return err
}

func (p *compactProtocolReader) readVarint() (int64, error) {
	if br, ok := p.r.(io.ByteReader); ok {
		return binary.ReadVarint(br)
	}
	// TODO: Make this more efficient
	n := 0
	b := p.buf
	for {
		if _, err := p.r.Read(b[n : n+1]); err != nil {
			return 0, err
		}
		n++
		// n == 0: buf too small
		// n  < 0: value larger than 64-bits
		if val, n := binary.Varint(b[:n]); n > 0 {
			return val, nil
		} else if n < 0 {
			return val, ProtocolError{"CompactProtocol", "varint overflow on read"}
		}
	}
}

func (p *compactProtocolReader) readUvarint() (uint64, error) {
	if br, ok := p.r.(io.ByteReader); ok {
		return binary.ReadUvarint(br)
	}
	// TODO: Make this more efficient
	n := 0
	b := p.buf
	for {
		if _, err := p.r.Read(b[n : n+1]); err != nil {
			return 0, err
		}
		n++
		// n == 0: buf too small
		// n  < 0: value larger than 64-bits
		if val, n := binary.Uvarint(b[:n]); n > 0 {
			return val, nil
		} else if n < 0 {
			return val, ProtocolError{"CompactProtocol", "varint overflow on read"}
		}
	}
}

func (p *compactProtocolReader) ReadMessageBegin() (string, byte, int32, error) {
	protocolID, err := p.ReadByte()
	if err != nil {
		return "", 0, -1, err
	}
	if protocolID != compactProtocolID {
		return "", 0, -1, ProtocolError{"CompactProtocol", "invalid compact protocol ID"}
	}
	versionAndType, err := p.ReadByte()
	if err != nil {
		return "", 0, -1, err
	}
	version := versionAndType & compactVersionMask
	if version != compactVersion {
		return "", 0, -1, ProtocolError{"CompactProtocol", "invalid compact protocol version"}
	}
	msgType := (versionAndType >> compactTypeShiftAmount) & 0x03
	seqID, err := p.readUvarint()
	if err != nil {
		return "", 0, -1, err
	}
	msgName, err := p.ReadString()
	if err != nil {
		return "", 0, -1, err
	}
	return msgName, msgType, int32(seqID), nil
}

// Read a struct begin. There's nothing on the wire for this, but it is our
// opportunity to push a new struct begin marker onto the field stack.
func (p *compactProtocolReader) ReadStructBegin() error {
	p.structs = append(p.structs, p.lastFieldID)
	p.lastFieldID = 0
	return nil
}

// Doesn't actually consume any wire data, just removes the last field for
// this struct from the field stack.
func (p *compactProtocolReader) ReadStructEnd() error {
	// consume the last field we read off the wire
	p.lastFieldID = p.structs[len(p.structs)-1]
	p.structs = p.structs[:len(p.structs)-1]
	return nil
}

// Read a field header off the wire.
func (p *compactProtocolReader) ReadFieldBegin() (byte, int16, error) {
	compactType, err := p.ReadByte()
	if err != nil {
		return 0, -1, err
	}

	// if it's a stop, then we can return immediately, as the struct is over
	if (compactType & 0x0f) == ctStop {
		return TypeStop, -1, nil
	}

	// mask off the 4 MSB of the type header. it could contain a field id delta.
	var fieldID int16
	modifier := int16((compactType & 0xf0) >> 4)
	if modifier == 0 {
		// not a delta. look ahead for the zigzag varint field id.
		fieldID, err = p.ReadI16()
		if err != nil {
			return 0, fieldID, err
		}
	} else {
		// has a delta. add the delta to the last read field id
		fieldID = p.lastFieldID + modifier
	}

	fieldType := compactTypeToThriftType[compactType&0x0f]

	// if this happens to be a boolean field, the value is encoded in the type
	if fieldType == TypeBool {
		// save the boolean value in a special instance variable.
		p.boolValue = (compactType & 0x0f) == ctTrue
		p.boolFid = fieldID
	}

	// push the new field onto the field stack so we can keep the deltas going.
	p.lastFieldID = fieldID
	return fieldType, fieldID, nil
}

// Read a map header off the wire. If the size is zero, skip reading the key
// and value type. This means that 0-length maps will yield TMaps without the
// "correct" types.
func (p *compactProtocolReader) ReadMapBegin() (byte, byte, int, error) {
	size, err := p.readUvarint()
	if err != nil {
		return 0, 0, -1, err
	}
	keyAndValueType := byte(0)
	if size > 0 {
		keyAndValueType, err = p.ReadByte()
		if err != nil {
			return 0, 0, -1, err
		}
	}
	return compactTypeToThriftType[keyAndValueType>>4], compactTypeToThriftType[keyAndValueType&0x0f], int(size), nil
}

// Read a list header off the wire. If the list size is 0-14, the size will
// be packed into the element type header. If it's a longer list, the 4 MSB
// of the element type header will be 0xF, and a varint will follow with the
// true size.
func (p *compactProtocolReader) ReadListBegin() (byte, int, error) {
	sizeAndType, err := p.ReadByte()
	if err != nil {
		return 0, -1, err
	}
	size := int((sizeAndType >> 4) & 0x0f)
	if size == 15 {
		s, err := p.readUvarint()
		if err != nil {
			return 0, -1, err
		}
		size = int(s)
	}
	return compactTypeToThriftType[sizeAndType&0x0f], size, nil
}

// Read a set header off the wire. If the set size is 0-14, the size will
// be packed into the element type header. If it's a longer set, the 4 MSB
// of the element type header will be 0xF, and a varint will follow with the
// true size.
func (p *compactProtocolReader) ReadSetBegin() (byte, int, error) {
	return p.ReadListBegin()
}

// Read a boolean off the wire. If this is a boolean field, the value should
// already have been read during readFieldBegin, so we'll just consume the
// pre-stored value. Otherwise, read a byte.
func (p *compactProtocolReader) ReadBool() (bool, error) {
	if p.boolFid < 0 {
		v, err := p.ReadByte()
		return v == ctTrue, err
	}

	res := p.boolValue
	p.boolFid = -1
	return res, nil
}

// Read a single byte off the wire. Nothing interesting here.
func (p *compactProtocolReader) ReadByte() (byte, error) {
	b := p.buf
	_, err := io.ReadFull(p.r, b[:1])
	return b[0], err
}

func (p *compactProtocolReader) ReadI16() (int16, error) {
	v, err := p.readVarint()
	return int16(v), err
}

func (p *compactProtocolReader) ReadI32() (int32, error) {
	v, err := p.readVarint()
	return int32(v), err
}

func (p *compactProtocolReader) ReadI64() (int64, error) {
	v, err := p.readVarint()
	return v, err
}

func (p *compactProtocolReader) ReadDouble() (float64, error) {
	b := p.buf
	_, err := io.ReadFull(p.r, b[:8])
	value := math.Float64frombits(binary.BigEndian.Uint64(b))
	return value, err
}

func (p *compactProtocolReader) ReadString() (string, error) {
	ln, err := p.readUvarint()
	if err != nil || ln == 0 {
		return "", err
	} else if ln < 0 {
		return "", ProtocolError{"CompactProtocol", "negative length in CompactProtocol.ReadString"}
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

func (p *compactProtocolReader) ReadBytes() ([]byte, error) {
	ln, err := p.readUvarint()
	if err != nil || ln == 0 {
		return nil, err
	} else if ln < 0 {
		return nil, ProtocolError{"CompactProtocol", "negative length in CompactProtocol.ReadBytes"}
	}
	b := make([]byte, ln)
	if _, err := io.ReadFull(p.r, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (p *compactProtocolReader) ReadMessageEnd() error {
	return nil
}

func (p *compactProtocolReader) ReadFieldEnd() error {
	return nil
}

func (p *compactProtocolReader) ReadMapEnd() error {
	return nil
}

func (p *compactProtocolReader) ReadListEnd() error {
	return nil
}

func (p *compactProtocolReader) ReadSetEnd() error {
	return nil
}
