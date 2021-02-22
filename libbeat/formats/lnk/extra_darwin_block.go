package lnk

import (
	"errors"

	"github.com/elastic/beats/v7/libbeat/formats/common"
)

func parseExtraDarwin(size uint32, data []byte) (*Darwin, error) {
	if size != 0x00000314 {
		return nil, errors.New("invalid extra darwin block size")
	}
	ansi := common.ReadString(data[8:268], 0)
	unicode := common.ReadUnicode(data[268:788], 0)
	return &Darwin{
		ANSI:    ansi,
		Unicode: unicode,
	}, nil
}
