package lnk

import "errors"

func parseExtraVistaAndAboveIDList(size uint32, data []byte) (*VistaAndAboveIDList, error) {
	if size < 0x0000000A {
		return nil, errors.New("invalid extra vista and above id list block size")
	}
	targets, err := parseTargetList(data[8:])
	if err != nil {
		return nil, err
	}
	return &VistaAndAboveIDList{
		Targets: targets,
	}, nil
}
