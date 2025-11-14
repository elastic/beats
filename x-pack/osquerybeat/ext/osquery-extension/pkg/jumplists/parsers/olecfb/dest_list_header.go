// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package olecfb

import (
	"encoding/binary"
	"fmt"
	"strings"
)

type DestListHeader struct {
	Version               int32
	NumberOfEntries       int32
	NumberOfPinnedEntries int32
	UnknownCounter        float32
	LastEntryNumber       int32
	Unknown1              int32
	LastRevisionNumber    int32
	Unknown2              int32
}

const DestListHeaderSize = 32

func (h *DestListHeader) String() string {
	sb := strings.Builder{}
	sb.WriteString("<Header: {")
	sb.WriteString(fmt.Sprintf("Version: %d, ", h.Version))
	sb.WriteString(fmt.Sprintf("NumberOfEntries: %d, ", h.NumberOfEntries))
	sb.WriteString(fmt.Sprintf("NumberOfPinnedEntries: %d, ", h.NumberOfPinnedEntries))
	sb.WriteString(fmt.Sprintf("UnknownCounter: %f, ", h.UnknownCounter))
	sb.WriteString(fmt.Sprintf("LastEntryNumber: %d, ", h.LastEntryNumber))
	sb.WriteString(fmt.Sprintf("Unknown1: %d, ", h.Unknown1))
	sb.WriteString(fmt.Sprintf("LastRevisionNumber: %d, ", h.LastRevisionNumber))
	sb.WriteString(fmt.Sprintf("Unknown2: %d", h.Unknown2))
	sb.WriteString("}")
	return sb.String()
}

func NewDestListHeader(data []byte) (*DestListHeader, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("data is too short to contain a DestListHeader")
	}
	header := &DestListHeader{
		Version:               int32(binary.LittleEndian.Uint32(data[0:4])),
		NumberOfEntries:       int32(binary.LittleEndian.Uint32(data[4:8])),
		NumberOfPinnedEntries: int32(binary.LittleEndian.Uint32(data[8:12])),
		UnknownCounter:        float32(binary.LittleEndian.Uint32(data[12:16])),
		LastEntryNumber:       int32(binary.LittleEndian.Uint32(data[16:20])),
		Unknown1:              int32(binary.LittleEndian.Uint32(data[20:24])),
		LastRevisionNumber:    int32(binary.LittleEndian.Uint32(data[24:28])),
		Unknown2:              int32(binary.LittleEndian.Uint32(data[28:32])),
	}
	return header, nil
}
