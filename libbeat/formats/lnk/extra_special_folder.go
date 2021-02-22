package lnk

import (
	"encoding/binary"
	"errors"
)

func parseExtraSpecialFolder(size uint32, data []byte) (*SpecialFolder, error) {
	if size != 0x00000010 {
		return nil, errors.New("invalid extra special folder block size")
	}
	return &SpecialFolder{
		ID:     binary.LittleEndian.Uint32(data[8:12]),
		Offset: binary.LittleEndian.Uint32(data[12:16]),
	}, nil
}
