package encoding

import (
	"errors"
	"io"
	"os"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type endianness int8

const (
	unknownEndianess endianness = iota
	bigEndian
	littleEndian
)

var ErrUnsupportedSourceTypeBOM = errors.New("source type not support by BOM based encoding")

// utf16 BOM based encodings. Only seekable data sources are supported for
// the need to check the optional Byte Order Marker being available in data source
// before configuring the actual decoder and encoder.
var (
	// BOM is required, as no fallback is specified
	utf16BOMRequired = utf16BOM(unknownEndianess)

	// BOM is optional. Falls back to BigEndian if missing
	utf16BOMBigEndian = utf16BOM(bigEndian)

	// BOM is optional. Falls back to LittleEndian if missing
	utf16BOMLittleEndian = utf16BOM(littleEndian)
)

var utf16Map = map[endianness]Encoding{
	bigEndian:    unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM),
	littleEndian: unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM),
}

func utf16BOM(e endianness) EncodingFactory {
	return func(in_ io.Reader) (Encoding, error) {
		in, ok := in_.(io.ReadSeeker)
		if !ok {
			return nil, ErrUnsupportedSourceTypeBOM
		}

		return utf16Seekable(in, e)
	}
}

func utf16Seekable(in io.ReadSeeker, endianness endianness) (Encoding, error) {
	// remember file offset in case we have to back off
	offset, err := in.Seek(0, os.SEEK_CUR)
	if err != nil {
		return nil, err
	}

	// goto beginning of file
	keepOffset := offset == 0
	if _, err = in.Seek(0, os.SEEK_SET); err != nil {
		return nil, err
	}

	// read Byte Order Marker (BOM)
	var buf [2]byte
	n, err := in.Read(buf[:])
	if err != nil {
		in.Seek(offset, os.SEEK_SET)
		return nil, err
	}
	if n < 2 {
		in.Seek(offset, os.SEEK_SET)
		return nil, transform.ErrShortSrc
	}

	// determine endianess from BOM
	inEndiannes := unknownEndianess
	switch {
	case buf[0] == 0xfe && buf[1] == 0xff:
		inEndiannes = bigEndian
	case buf[0] == 0xff && buf[1] == 0xfe:
		inEndiannes = littleEndian
	}

	// restore offset if BOM is missing or this function was not
	// called with read pointer at beginning of file
	if !keepOffset || inEndiannes == unknownEndianess {
		if _, err = in.Seek(offset, os.SEEK_SET); err != nil {
			return nil, err
		}
	}

	// choose encoding based on BOM
	if encoding, ok := utf16Map[inEndiannes]; ok {
		return encoding, nil
	}

	// fall back to configured endianess
	if encoding, ok := utf16Map[endianness]; ok {
		return encoding, nil
	}

	// no encoding for configured endianess found => fail
	return nil, unicode.ErrMissingBOM
}
