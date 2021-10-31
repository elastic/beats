package sys

import (
	"encoding/binary"
	"unsafe"
)

func GetEndian() binary.ByteOrder {
	var i int32 = 0x1
	v := (*[4]byte)(unsafe.Pointer(&i))
	if v[0] == 0 {
		return binary.BigEndian
	} else {
		return binary.LittleEndian
	}
}
