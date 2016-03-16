package sys

import (
	"fmt"
	"strings"
	"unicode/utf16"
)

// UTF16BytesToString returns the Unicode code point sequence represented
// by the UTF-16 buffer b.
func UTF16BytesToString(b []byte) (string, int, error) {
	if len(b)%2 != 0 {
		return "", 0, fmt.Errorf("Slice must have an even length (length=%d)",
			len(b))
	}

	offset := len(b)/2 + 2
	s := make([]uint16, len(b)/2)
	for i := range s {
		s[i] = uint16(b[i*2]) + uint16(b[(i*2)+1])<<8

		if s[i] == 0 {
			s = s[0:i]
			offset = i*2 + 2
			break
		}
	}

	return string(utf16.Decode(s)), offset, nil
}

// RemoveWindowsLineEndings replaces carriage return line feed (CRLF) with
// line feed (LF) and trims any newline character that may exist at the end
// of the string.
func RemoveWindowsLineEndings(s string) string {
	s = strings.Replace(s, "\r\n", "\n", -1)
	return strings.TrimRight(s, "\n")
}
