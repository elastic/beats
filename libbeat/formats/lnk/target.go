package lnk

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"

	sha256 "github.com/minio/sha256-simd"
)

func parseTargets(header *Header, offset int64, r io.ReaderAt) ([]Target, int64, error) {
	if !hasFlag(header.rawLinkFlags, hasTargetIDList) {
		return nil, offset, nil
	}

	sizeData := make([]byte, 2)
	n, err := r.ReadAt(sizeData, offset)
	if err != nil {
		return nil, 0, err
	}
	if n != 2 {
		return nil, 0, errors.New("invalid target list")
	}
	offset += 2
	size := binary.LittleEndian.Uint16(sizeData)
	data := make([]byte, size)
	n, err = r.ReadAt(data, offset)
	if err != nil {
		return nil, 0, err
	}
	if n != int(size) {
		return nil, 0, errors.New("invalid target list size")
	}
	targets, err := parseTargetList(data)
	return targets, offset + int64(size), err
}

func parseTargetList(data []byte) ([]Target, error) {
	// https://github.com/libyal/libfwsi/blob/master/documentation/Windows%20Shell%20Item%20format.asciidoc#2-shell-item-list
	targets := []Target{}
	offset := 0
	for {
		targetData := data[offset:]
		if len(targetData) < 3 {
			// early end
			return targets, nil
		}
		targetSize := binary.LittleEndian.Uint16(targetData[0:2])
		if targetSize == 0 {
			return targets, nil
		}
		if len(targetData) < int(targetSize) {
			// we have an invalid target
			return targets, nil
		}
		targetData = targetData[:targetSize]
		targetType := targetData[3]
		hash := sha256.Sum256(targetData[4:])
		targets = append(targets, Target{
			Size:   targetSize,
			TypeID: targetType,
			SHA256: hex.EncodeToString(hash[:]),
		})
		offset += int(targetSize)
	}
}
