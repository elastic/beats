package lnk

import (
	"errors"

	"github.com/elastic/beats/v7/libbeat/formats/common"
)

func parseExtraShim(size uint32, data []byte) (*Shim, error) {
	if size < 0x00000088 {
		return nil, errors.New("invalid extra shim block size")
	}
	return &Shim{
		LayerName: common.ReadUnicode(data, 8),
	}, nil
}
