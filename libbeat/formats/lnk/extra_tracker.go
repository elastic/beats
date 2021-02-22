package lnk

import (
	"encoding/binary"
	"errors"

	"github.com/elastic/beats/v7/libbeat/formats/common"
)

func parseExtraTracker(size uint32, data []byte) (*Tracker, error) {
	if size != 0x00000060 {
		return nil, errors.New("invalid extra tracker block size")
	}
	return &Tracker{
		Version:   binary.LittleEndian.Uint32(data[12:16]),
		MachineID: common.ReadString(data[16:32], 0),
		Droid: []string{
			encodeUUID(data[32:48]),
			encodeUUID(data[48:64]),
		},
		DroidBirth: []string{
			encodeUUID(data[64:80]),
			encodeUUID(data[80:96]),
		},
	}, nil
}
