package rpm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// A Header stores metadata about a rpm package.
type Header struct {
	Version    int
	IndexCount int
	Length     int
	Indexes    IndexEntries
	Start      int
	End        int
}

// Headers is an array of Header structs.
type Headers []Header

// Predefined sizing constraints.
const (
	// MAX_HEADER_SIZE is the maximum allowable header size in bytes (32 MB).
	MAX_HEADER_SIZE = 33554432
)

// Predefined header errors.
var (
	// ErrBadHeaderLength indicates that the read header section is not the
	// expected length.
	ErrBadHeaderLength = fmt.Errorf("RPM header section is incorrect length")

	// ErrNotHeader indicates that the read header section does start with the
	// expected descriptor.
	ErrNotHeader = fmt.Errorf("invalid RPM header descriptor")

	// ErrBadStoreLength indicates that the read header store section is not the
	// expected length.
	ErrBadStoreLength = fmt.Errorf("header value store is incorrect length")
)

// Predefined header index errors.
var (
	// ErrBadIndexCount indicates that number of indexes given in the read
	// header would exceed the actual size of the header.
	ErrBadIndexCount = fmt.Errorf("index count exceeds header size")

	// ErrBadIndexLength indicates that the read header index section is not the
	// expected length.
	ErrBadIndexLength = fmt.Errorf("index section is incorrect length")

	// ErrIndexOutOfRange indicates that the read header index would exceed the
	// range of the header.
	ErrIndexOutOfRange = fmt.Errorf("index is out of range")

	// ErrBadIndexType indicates that the read index contains a value of an
	// unsupported data type.
	ErrBadIndexType = fmt.Errorf("unknown index data type")

	// ErrBadIndexValueCount indicates that the read index value would exceed
	// the range of the header store section.
	ErrBadIndexValueCount = fmt.Errorf("index value count is out of range")
)

// ReadPackageHeader reads an RPM package file header structure from the given
// io.Reader.
//
// This function should only be used if you intend to read a package header
// structure in isolation.
func ReadPackageHeader(r io.Reader) (*Header, error) {
	// read the "header structure header"
	header := make([]byte, 16)
	_, err := io.ReadFull(r, header)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil, ErrBadHeaderLength
		}

		return nil, err
	}

	// check magic number
	if 0 != bytes.Compare(header[:3], []byte{0x8E, 0xAD, 0xE8}) {
		return nil, ErrNotHeader
	}

	// translate header
	h := &Header{
		Version:    int(header[3]),
		IndexCount: int(binary.BigEndian.Uint32(header[8:12])),
		Length:     int(binary.BigEndian.Uint32(header[12:16])),
	}

	// make sure header size is in range
	if h.Length > MAX_HEADER_SIZE {
		return nil, ErrBadHeaderLength
	}

	// Ensure index count is in range
	// This test is not entirely precise as h.Length also includes the value
	// store. It should at least help eliminate excessive buffer allocations for
	// corrupted length values in the > h.Length ranges.
	if h.IndexCount*16 > h.Length {
		return nil, ErrBadIndexCount
	}

	h.Indexes = make(IndexEntries, h.IndexCount)

	// read indexes
	indexLength := 16 * h.IndexCount
	indexes := make([]byte, indexLength)
	_, err = io.ReadFull(r, indexes)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil, ErrBadIndexLength
		}
		return nil, err
	}

	for x := 0; x < h.IndexCount; x++ {
		o := 16 * x
		index := IndexEntry{
			Tag:       int(binary.BigEndian.Uint32(indexes[o : o+4])),
			Type:      int(binary.BigEndian.Uint32(indexes[o+4 : o+8])),
			Offset:    int(binary.BigEndian.Uint32(indexes[o+8 : o+12])),
			ItemCount: int(binary.BigEndian.Uint32(indexes[o+12 : o+16])),
		}

		// validate index offset
		if index.Offset >= h.Length {
			return nil, ErrIndexOutOfRange
		}

		// append
		h.Indexes[x] = index
	}

	// read the "store"
	store := make([]byte, h.Length)
	_, err = io.ReadFull(r, store)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil, ErrBadStoreLength
		}

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
				if o >= len(store) {
					return nil, fmt.Errorf("uint8 value for index %d is out of range", x+1)
				}

				vals[v] = uint8(store[o])
				o += 1
			}

			index.Value = vals

		case IndexDataTypeInt8:
			vals := make([]int8, index.ItemCount)
			for v := 0; v < index.ItemCount; v++ {
				if o >= len(store) {
					return nil, fmt.Errorf("int8 value for index %d is out of range", x+1)
				}

				vals[v] = int8(store[o])
				o += 1
			}

			index.Value = vals

		case IndexDataTypeInt16:
			vals := make([]int16, index.ItemCount)
			for v := 0; v < index.ItemCount; v++ {
				if o+2 > len(store) {
					return nil, fmt.Errorf("int16 value for index %d is out of range", x+1)
				}

				vals[v] = int16(binary.BigEndian.Uint16(store[o : o+2]))
				o += 2
			}

			index.Value = vals

		case IndexDataTypeInt32:
			vals := make([]int32, index.ItemCount)
			for v := 0; v < index.ItemCount; v++ {
				if o+4 > len(store) {
					return nil, fmt.Errorf("int32 value for index %d is out of range", x+1)
				}

				vals[v] = int32(binary.BigEndian.Uint32(store[o : o+4]))
				o += 4
			}

			index.Value = vals

		case IndexDataTypeInt64:
			vals := make([]int64, index.ItemCount)
			for v := 0; v < index.ItemCount; v++ {
				if o+8 > len(store) {
					return nil, fmt.Errorf("int64 value for index %d is out of range", x+1)
				}

				vals[v] = int64(binary.BigEndian.Uint64(store[o : o+8]))
				o += 8
			}

			index.Value = vals

		case IndexDataTypeBinary:
			if o+index.ItemCount > len(store) {
				return nil, fmt.Errorf("[]byte value for index %d is out of range", x+1)
			}

			b := make([]byte, index.ItemCount)
			copy(b, store[o:o+index.ItemCount])

			index.Value = b

		case IndexDataTypeString, IndexDataTypeStringArray, IndexDataTypeI8NString:
			// allow atleast one byte per string
			if o+index.ItemCount > len(store) {
				return nil, fmt.Errorf("[]string value for index %d is out of range", x+1)
			}

			vals := make([]string, index.ItemCount)

			for s := 0; s < index.ItemCount; s++ {
				// calculate string length
				var j int
				for j = 0; (o+j) < len(store) && store[o+j] != 0; j++ {
				}

				if j == len(store) {
					return nil, fmt.Errorf("string value for index %d is out of range", x+1)
				}

				vals[s] = string(store[o : o+j])
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

	// calculate location of the end of the header by padding to a multiple of 8
	o := 8 - int(math.Mod(float64(h.Length), 8))

	// seek to the end of the header
	if o > 0 && o < 8 {
		pad := make([]byte, o)
		_, err = io.ReadFull(r, pad)
		if err != nil {
			return nil, fmt.Errorf("Error seeking beyond header padding of %d bytes: %v", o, err)
		}
	}

	return h, nil
}
