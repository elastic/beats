// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package olecfb

import (
	"encoding/binary"
	"strings"
	"fmt"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

type DestList struct {
	Header DestListHeader
	Entries []DestListEntry
}

func (d *DestList) String() string {
	sb := strings.Builder{}
	sb.WriteString("<DestList: {")
	sb.WriteString(fmt.Sprintf("Header: %s, ", d.Header.String()))
	sb.WriteString("Entries: [")
	for _, entry := range d.Entries {
		sb.WriteString(fmt.Sprintf("%s, ", entry.String()))
	}
	sb.WriteString("]")
	sb.WriteString("}")
	return sb.String()
}

//https://github.com/EricZimmerman/JumpList/blob/master/JumpList/Automatic/DestList.cs#L9
func NewDestList(data []byte, log *logger.Logger) (*DestList, error) {
	if len(data) < DestListHeaderSize {
		return nil, fmt.Errorf("data is too short to contain a DestList header")
	}
	
	header, err := NewDestListHeader(data[:DestListHeaderSize])
	if err != nil {
		return nil, fmt.Errorf("failed to parse DestList header: %w", err)
	}

    destList := &DestList{
		Header: *header,
		Entries: make([]DestListEntry, 0),
	}

    index := DestListHeaderSize

	if header.Version == 1 {
		for i := 0; i < int(destList.Header.NumberOfEntries); i++ {
			pathSize := binary.LittleEndian.Uint16(data[index+112:])
			entrySize := 112 + 2 + (int(pathSize) * 2)
			entryBytes := data[index:index+entrySize]
			entry, err := NewDestListEntry(entryBytes, header.Version)
			if err != nil {
				return nil, fmt.Errorf("failed to parse DestList entry: %w", err)
			}
			destList.Entries = append(destList.Entries, *entry)
			index += entrySize
		}
	} else {
		for i := 0; i < int(destList.Header.NumberOfEntries); i++ {
			pathSize := binary.LittleEndian.Uint16(data[index+128:])
			entrySize := 128 + 2 + (int(pathSize) * 2)
			spsSize := binary.LittleEndian.Uint32(data[:index + entrySize])
			entryStart := index
			entryEnd := entryStart + entrySize + int(spsSize)
			if entryEnd > len(data) {
				return nil, fmt.Errorf("entry end is out of bounds")
			}
			entryBytes := data[entryStart:entryEnd]
			entry, err := NewDestListEntry(entryBytes, header.Version)
			if err != nil {
				return nil, fmt.Errorf("failed to parse DestList entry: %w", err)
			}
			destList.Entries = append(destList.Entries, *entry)
			index = entryEnd
		}
	}
	return destList, nil
}