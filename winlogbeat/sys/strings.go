package sys

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// The conditions replacementChar==unicode.ReplacementChar and
// maxRune==unicode.MaxRune are verified in the tests.
// Defining them locally avoids this package depending on package unicode.

const (
	replacementChar = '\uFFFD'     // Unicode replacement character
	maxRune         = '\U0010FFFF' // Maximum valid Unicode code point.
)

const (
	// 0xd800-0xdc00 encodes the high 10 bits of a pair.
	// 0xdc00-0xe000 encodes the low 10 bits of a pair.
	// the value is those 20 bits plus 0x10000.
	surr1 = 0xd800
	surr2 = 0xdc00
	surr3 = 0xe000

	surrSelf = 0x10000
)

var ErrBufferTooSmall = errors.New("buffer too small")

func UTF16ToUTF8Bytes(in []byte, out io.Writer) error {
	if len(in)%2 != 0 {
		return fmt.Errorf("input buffer must have an even length (length=%d)", len(in))
	}

	var runeBuf [4]byte
	var v1, v2 uint16
	for i := 0; i < len(in); i += 2 {
		v1 = uint16(in[i]) | uint16(in[i+1])<<8
		// Stop at null-terminator.
		if v1 == 0 {
			return nil
		}

		switch {
		case v1 < surr1, surr3 <= v1:
			n := utf8.EncodeRune(runeBuf[:], rune(v1))
			out.Write(runeBuf[:n])
		case surr1 <= v1 && v1 < surr2 && len(in) > i+2:
			v2 = uint16(in[i+2]) | uint16(in[i+3])<<8
			if surr2 <= v2 && v2 < surr3 {
				// valid surrogate sequence
				r := utf16.DecodeRune(rune(v1), rune(v2))
				n := utf8.EncodeRune(runeBuf[:], r)
				out.Write(runeBuf[:n])
			}
			i += 2
		default:
			// invalid surrogate sequence
			n := utf8.EncodeRune(runeBuf[:], replacementChar)
			out.Write(runeBuf[:n])
		}
	}

	return nil
}

// UTF16BytesToString returns a string that is decoded from the UTF-16 bytes.
// The byte slice must be of even length otherwise an error will be returned.
// The integer returned is the offset to the start of the next string with
// buffer if it exists, otherwise -1 is returned.
func UTF16BytesToString(b []byte) (string, int, error) {
	if len(b)%2 != 0 {
		return "", 0, fmt.Errorf("Slice must have an even length (length=%d)", len(b))
	}

	offset := -1

	// Find the null terminator if it exists and re-slice the b.
	if nullIndex := indexNullTerminator(b); nullIndex > -1 {
		if len(b) > nullIndex+2 {
			offset = nullIndex + 2
		}

		b = b[:nullIndex]
	}

	s := make([]uint16, len(b)/2)
	for i := range s {
		s[i] = uint16(b[i*2]) + uint16(b[(i*2)+1])<<8
	}

	return string(utf16.Decode(s)), offset, nil
}

// indexNullTerminator returns the index of a null terminator within a buffer
// containing UTF-16 encoded data. If the null terminator is not found -1 is
// returned.
func indexNullTerminator(b []byte) int {
	if len(b) < 2 {
		return -1
	}

	for i := 0; i < len(b); i += 2 {
		if b[i] == 0 && b[i+1] == 0 {
			return i
		}
	}

	return -1
}

// RemoveWindowsLineEndings replaces carriage return line feed (CRLF) with
// line feed (LF) and trims any newline character that may exist at the end
// of the string.
func RemoveWindowsLineEndings(s string) string {
	s = strings.Replace(s, "\r\n", "\n", -1)
	return strings.TrimRight(s, "\n")
}
