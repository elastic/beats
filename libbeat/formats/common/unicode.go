package common

import (
	"encoding/binary"
	"unicode/utf16"
)

// ReadUnicode decodes a unicode string ending with a null
func ReadUnicode(data []byte, offset int) string {
	encode := []uint16{}
	for {
		if len(data) < offset+1 {
			return string(utf16.Decode(encode))
		}
		value := binary.LittleEndian.Uint16(data[offset : offset+2])
		if value == 0 {
			return string(utf16.Decode(encode))
		}
		encode = append(encode, value)
		offset += 2
	}
}
