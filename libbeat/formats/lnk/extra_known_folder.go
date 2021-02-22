package lnk

import (
	"encoding/binary"
	"errors"
)

func parseExtraKnownFolder(size uint32, data []byte) (*KnownFolder, error) {
	if size != 0x0000001C {
		return nil, errors.New("invalid extra known folder block size")
	}
	return &KnownFolder{
		ID:     encodeUUID(data[8:24]),
		Offset: binary.LittleEndian.Uint32(data[24:28]),
	}, nil
}
