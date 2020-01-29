package rpm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// A Header stores metadata about a rpm package.
type Header struct {
	Version    int
	IndexCount int
	Length     int
	Indexes    IndexEntries
}

const (
	r_MaxHeaderSize      = 33554432
	r_HeaderHeaderLength = 16
	r_IndexHeaderLength  = 16
)

var (
	// ErrBadHeaderLength indicates that the read header section is not the
	// expected length.
	ErrBadHeaderLength = errors.New("RPM header section is incorrect length")

	// ErrNotHeader indicates that the read header section does start with the
	// expected descriptor.
	ErrNotHeader = errors.New("invalid RPM header descriptor")

	// ErrBadIndexCount indicates that number of indexes given in the read
	// header would exceed the actual size of the header.
	ErrBadIndexCount = errors.New("index count exceeds header size")

	// ErrIndexOutOfRange indicates that the read header index would exceed the
	// range of the header.
	ErrIndexOutOfRange = errors.New("index is out of range")

	// ErrBadIndexType indicates that the read index contains a value of an
	// unsupported data type.
	ErrBadIndexType = errors.New("unknown index data type")

	// ErrBadIndexValueCount indicates that the read index value would exceed
	// the range of the header store section.
	ErrBadIndexValueCount = errors.New("index value count is out of range")
)

// ReadPackageHeader reads an RPM package file header structure from the given
// io.Reader.
//
// This function should only be used if you intend to read a package header
// structure in isolation.
func ReadPackageHeader(r io.Reader) (*Header, error) {
	buf := make([]byte, r_HeaderHeaderLength)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}
	if 0 != bytes.Compare(buf[:3], []byte{0x8E, 0xAD, 0xE8}) {
		return nil, ErrNotHeader
	}
	h := &Header{
		Version:    int(buf[3]),
		IndexCount: int(binary.BigEndian.Uint32(buf[8:12])),
		Length:     int(binary.BigEndian.Uint32(buf[12:16])),
	}
	if h.Length > r_MaxHeaderSize {
		return nil, ErrBadHeaderLength
	}
	if h.IndexCount*r_IndexHeaderLength > r_MaxHeaderSize {
		return nil, ErrBadIndexCount
	}

	// read indexes
	h.Indexes = make(IndexEntries, h.IndexCount)
	indexLength := r_IndexHeaderLength * h.IndexCount
	buf = make([]byte, indexLength)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}
	for x := 0; x < h.IndexCount; x++ {
		ib := buf[r_IndexHeaderLength*x:]
		h.Indexes[x] = IndexEntry{
			Tag:       int(binary.BigEndian.Uint32(ib[:4])),
			Type:      int(binary.BigEndian.Uint32(ib[4:8])),
			Offset:    int(binary.BigEndian.Uint32(ib[8:12])),
			ItemCount: int(binary.BigEndian.Uint32(ib[12:16])),
		}
		if h.Indexes[x].Offset >= h.Length {
			return nil, ErrIndexOutOfRange
		}
	}

	// read the "store"
	buf = make([]byte, h.Length)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}

	// parse the value of each index from the store
	for x := 0; x < h.IndexCount; x++ {
		index := h.Indexes[x]
		o := index.Offset
		if index.ItemCount == 0 {
			return nil, ErrBadIndexValueCount
		}

		switch index.Type {
		case IndexDataTypeChar:
			vals := make([]uint8, index.ItemCount)
			for v := 0; v < index.ItemCount; v++ {
				if o >= len(buf) {
					return nil, fmt.Errorf("uint8 value for index %d is out of range", x+1)
				}

				vals[v] = uint8(buf[o])
				o++
			}

			index.Value = vals

		case IndexDataTypeInt8:
			vals := make([]int8, index.ItemCount)
			for v := 0; v < index.ItemCount; v++ {
				if o >= len(buf) {
					return nil, fmt.Errorf("int8 value for index %d is out of range", x+1)
				}

				vals[v] = int8(buf[o])
				o++
			}

			index.Value = vals

		case IndexDataTypeInt16:
			vals := make([]int16, index.ItemCount)
			for v := 0; v < index.ItemCount; v++ {
				if o+2 > len(buf) {
					return nil, fmt.Errorf("int16 value for index %d is out of range", x+1)
				}

				vals[v] = int16(binary.BigEndian.Uint16(buf[o : o+2]))
				o += 2
			}

			index.Value = vals

		case IndexDataTypeInt32:
			vals := make([]int32, index.ItemCount)
			for v := 0; v < index.ItemCount; v++ {
				if o+4 > len(buf) {
					return nil, fmt.Errorf("int32 value for index %d is out of range", x+1)
				}

				vals[v] = int32(binary.BigEndian.Uint32(buf[o : o+4]))
				o += 4
			}

			index.Value = vals

		case IndexDataTypeInt64:
			vals := make([]int64, index.ItemCount)
			for v := 0; v < index.ItemCount; v++ {
				if o+8 > len(buf) {
					return nil, fmt.Errorf("int64 value for index %d is out of range", x+1)
				}

				vals[v] = int64(binary.BigEndian.Uint64(buf[o : o+8]))
				o += 8
			}

			index.Value = vals

		case IndexDataTypeBinary:
			if o+index.ItemCount > len(buf) {
				return nil, fmt.Errorf("[]byte value for index %d is out of range", x+1)
			}

			b := make([]byte, index.ItemCount)
			copy(b, buf[o:o+index.ItemCount])

			index.Value = b

		case IndexDataTypeString, IndexDataTypeStringArray, IndexDataTypeI8NString:
			// allow at least one byte per string
			if o+index.ItemCount > len(buf) {
				return nil, fmt.Errorf("[]string value for index %d is out of range", x+1)
			}

			vals := make([]string, index.ItemCount)

			for s := 0; s < index.ItemCount; s++ {
				// calculate string length
				var j int
				for j = 0; (o+j) < len(buf) && buf[o+j] != 0; j++ {
				}

				if j == len(buf) {
					return nil, fmt.Errorf("string value for index %d is out of range", x+1)
				}

				vals[s] = string(buf[o : o+j])
				o += j + 1
			}

			index.Value = vals

		case IndexDataTypeNull:
		// nothing to do here

		default:
			// unknown data type
			return nil, ErrBadIndexType
		}

		// save in array
		h.Indexes[x] = index
	}

	return h, nil
}
