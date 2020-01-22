// This file has been generated from 'bin.yml', do not edit
package bin

import "encoding/binary"

// I8be wraps a byte array into a big endian encoded 8bit signed integer.
type I8be [1]byte

// Len returns the number of bytes required to store the value.
func (b *I8be) Len() int { return 1 }

// Get returns the decoded value.
func (b *I8be) Get() int8 {
	return int8(b[0])
}

// Set encodes a new value into the backing buffer:
func (b *I8be) Set(v int8) {
	b[0] = byte(v)
}

// I16be wraps a byte array into a big endian encoded 16bit signed integer.
type I16be [2]byte

// Len returns the number of bytes required to store the value.
func (b *I16be) Len() int { return 2 }

// Get returns the decoded value.
func (b *I16be) Get() int16 {
	return int16(binary.BigEndian.Uint16(b[:]))
}

// Set encodes a new value into the backing buffer:
func (b *I16be) Set(v int16) {
	binary.BigEndian.PutUint16(b[:], uint16(v))
}

// I32be wraps a byte array into a big endian encoded 32bit signed integer.
type I32be [4]byte

// Len returns the number of bytes required to store the value.
func (b *I32be) Len() int { return 4 }

// Get returns the decoded value.
func (b *I32be) Get() int32 {
	return int32(binary.BigEndian.Uint32(b[:]))
}

// Set encodes a new value into the backing buffer:
func (b *I32be) Set(v int32) {
	binary.BigEndian.PutUint32(b[:], uint32(v))
}

// I64be wraps a byte array into a big endian encoded 64bit signed integer.
type I64be [8]byte

// Len returns the number of bytes required to store the value.
func (b *I64be) Len() int { return 8 }

// Get returns the decoded value.
func (b *I64be) Get() int64 {
	return int64(binary.BigEndian.Uint64(b[:]))
}

// Set encodes a new value into the backing buffer:
func (b *I64be) Set(v int64) {
	binary.BigEndian.PutUint64(b[:], uint64(v))
}

// U8be wraps a byte array into a big endian encoded 8bit unsigned integer.
type U8be [1]byte

// Len returns the number of bytes required to store the value.
func (b *U8be) Len() int { return 1 }

// Get returns the decoded value.
func (b *U8be) Get() uint8 {
	return uint8(b[0])
}

// Set encodes a new value into the backing buffer:
func (b *U8be) Set(v uint8) {
	b[0] = byte(v)
}

// U16be wraps a byte array into a big endian encoded 16bit unsigned integer.
type U16be [2]byte

// Len returns the number of bytes required to store the value.
func (b *U16be) Len() int { return 2 }

// Get returns the decoded value.
func (b *U16be) Get() uint16 {
	return uint16(binary.BigEndian.Uint16(b[:]))
}

// Set encodes a new value into the backing buffer:
func (b *U16be) Set(v uint16) {
	binary.BigEndian.PutUint16(b[:], uint16(v))
}

// U32be wraps a byte array into a big endian encoded 32bit unsigned integer.
type U32be [4]byte

// Len returns the number of bytes required to store the value.
func (b *U32be) Len() int { return 4 }

// Get returns the decoded value.
func (b *U32be) Get() uint32 {
	return uint32(binary.BigEndian.Uint32(b[:]))
}

// Set encodes a new value into the backing buffer:
func (b *U32be) Set(v uint32) {
	binary.BigEndian.PutUint32(b[:], uint32(v))
}

// U64be wraps a byte array into a big endian encoded 64bit unsigned integer.
type U64be [8]byte

// Len returns the number of bytes required to store the value.
func (b *U64be) Len() int { return 8 }

// Get returns the decoded value.
func (b *U64be) Get() uint64 {
	return uint64(binary.BigEndian.Uint64(b[:]))
}

// Set encodes a new value into the backing buffer:
func (b *U64be) Set(v uint64) {
	binary.BigEndian.PutUint64(b[:], uint64(v))
}

// I8le wraps a byte array into a little endian encoded 8bit signed integer.
type I8le [1]byte

// Len returns the number of bytes required to store the value.
func (b *I8le) Len() int { return 1 }

// Get returns the decoded value.
func (b *I8le) Get() int8 {
	return int8(b[0])
}

// Set encodes a new value into the backing buffer:
func (b *I8le) Set(v int8) {
	b[0] = byte(v)
}

// I16le wraps a byte array into a little endian encoded 16bit signed integer.
type I16le [2]byte

// Len returns the number of bytes required to store the value.
func (b *I16le) Len() int { return 2 }

// Get returns the decoded value.
func (b *I16le) Get() int16 {
	return int16(binary.LittleEndian.Uint16(b[:]))
}

// Set encodes a new value into the backing buffer:
func (b *I16le) Set(v int16) {
	binary.LittleEndian.PutUint16(b[:], uint16(v))
}

// I32le wraps a byte array into a little endian encoded 32bit signed integer.
type I32le [4]byte

// Len returns the number of bytes required to store the value.
func (b *I32le) Len() int { return 4 }

// Get returns the decoded value.
func (b *I32le) Get() int32 {
	return int32(binary.LittleEndian.Uint32(b[:]))
}

// Set encodes a new value into the backing buffer:
func (b *I32le) Set(v int32) {
	binary.LittleEndian.PutUint32(b[:], uint32(v))
}

// I64le wraps a byte array into a little endian encoded 64bit signed integer.
type I64le [8]byte

// Len returns the number of bytes required to store the value.
func (b *I64le) Len() int { return 8 }

// Get returns the decoded value.
func (b *I64le) Get() int64 {
	return int64(binary.LittleEndian.Uint64(b[:]))
}

// Set encodes a new value into the backing buffer:
func (b *I64le) Set(v int64) {
	binary.LittleEndian.PutUint64(b[:], uint64(v))
}

// U8le wraps a byte array into a little endian encoded 8bit unsigned integer.
type U8le [1]byte

// Len returns the number of bytes required to store the value.
func (b *U8le) Len() int { return 1 }

// Get returns the decoded value.
func (b *U8le) Get() uint8 {
	return uint8(b[0])
}

// Set encodes a new value into the backing buffer:
func (b *U8le) Set(v uint8) {
	b[0] = byte(v)
}

// U16le wraps a byte array into a little endian encoded 16bit unsigned integer.
type U16le [2]byte

// Len returns the number of bytes required to store the value.
func (b *U16le) Len() int { return 2 }

// Get returns the decoded value.
func (b *U16le) Get() uint16 {
	return uint16(binary.LittleEndian.Uint16(b[:]))
}

// Set encodes a new value into the backing buffer:
func (b *U16le) Set(v uint16) {
	binary.LittleEndian.PutUint16(b[:], uint16(v))
}

// U32le wraps a byte array into a little endian encoded 32bit unsigned integer.
type U32le [4]byte

// Len returns the number of bytes required to store the value.
func (b *U32le) Len() int { return 4 }

// Get returns the decoded value.
func (b *U32le) Get() uint32 {
	return uint32(binary.LittleEndian.Uint32(b[:]))
}

// Set encodes a new value into the backing buffer:
func (b *U32le) Set(v uint32) {
	binary.LittleEndian.PutUint32(b[:], uint32(v))
}

// U64le wraps a byte array into a little endian encoded 64bit unsigned integer.
type U64le [8]byte

// Len returns the number of bytes required to store the value.
func (b *U64le) Len() int { return 8 }

// Get returns the decoded value.
func (b *U64le) Get() uint64 {
	return uint64(binary.LittleEndian.Uint64(b[:]))
}

// Set encodes a new value into the backing buffer:
func (b *U64le) Set(v uint64) {
	binary.LittleEndian.PutUint64(b[:], uint64(v))
}
