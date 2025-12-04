// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/richardlehane/mscfb"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

// DestListStreamName is the name of the stream that contains the destination list.
const DestListStreamName = "DestList"

// DestListPropertyStoreStreamName is the name of the stream that contains the destination list property store.
const DestListPropertyStoreStreamName = "DestListPropertyStore"

// DestListHeaderSize is the size of the DestList header.
const DestListHeaderSize = 32

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

type DestListEntry struct {
	MruPosition      int32
	Checksum         uint64
	VolumeDroid      *GUID
	FileDroid        *GUID
	VolumeBirthDroid *GUID
	FileBirthDroid   *GUID
	Hostname         string
	EntryNumber      int32
	Unknown0         int32
	AccessCount      float32
	LastModifiedTime time.Time
	PinStatus        int32
	InteractionCount int32
	Unknown3         int32
	Unknown4         int32
	Path             string
	MacAddress       string
	CreationTime     time.Time
	Lnk              *Lnk
	StreamName       string
}

type DestList struct {
	Header  DestListHeader
	Entries []*DestListEntry
}

// https://github.com/EricZimmerman/JumpList/blob/master/JumpList/Automatic/DestList.cs#L9
func NewDestList(data []byte, log *logger.Logger) (*DestList, error) {
	if len(data) < DestListHeaderSize {
		return nil, fmt.Errorf("data is too short to contain a DestList header")
	}

	header, err := NewDestListHeader(data[:DestListHeaderSize])
	if err != nil {
		return nil, fmt.Errorf("failed to parse DestList header: %w", err)
	}

	destList := &DestList{
		Header:  *header,
		Entries: make([]*DestListEntry, 0),	
	}

	index := DestListHeaderSize

	if header.Version == 1 {
		for i := 0; i < int(destList.Header.NumberOfEntries); i++ {
			pathSize := binary.LittleEndian.Uint16(data[index+112:])
			entrySize := 112 + 2 + (int(pathSize) * 2)
			entryBytes := data[index : index+entrySize]
			entry, err := NewDestListEntry(entryBytes, header.Version)
			if err != nil {
				return nil, fmt.Errorf("failed to parse DestList entry: %w", err)
			}
			destList.Entries = append(destList.Entries, entry)
			index += entrySize
		}
	} else {
		for i := 0; i < int(destList.Header.NumberOfEntries); i++ {
			pathSize := binary.LittleEndian.Uint16(data[index+128:])
			entrySize := 128 + 2 + (int(pathSize) * 2)
			spsSize := binary.LittleEndian.Uint32(data[:index+entrySize])
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
			destList.Entries = append(destList.Entries, entry)
			index = entryEnd
		}
	}
	return destList, nil
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

func parseHostname(data []byte) string {
	if len(data) < 16 {
		return ""
	}

	findNullIndex := func(data any) int {
		switch s := data.(type) {
		case []uint16:
			for i, char := range s {
				if char == 0 {
					return i
				}
			}
		case []byte:
			for i, char := range s {
				if char == 0 {
					return i
				}
			}
		}
		return 0
	}

	if data[1] == 0 {
		utf16data := make([]uint16, 8)
		i := 0
		for i < 8 {
			start := i * 2
			end := start + 2
			utf16data[i] = binary.LittleEndian.Uint16(data[start:end])
			i++
		}
		nullIndex := findNullIndex(utf16data)
		return string(utf16.Decode(utf16data[:nullIndex]))
	} else {
		nullIndex := findNullIndex(data)
		return string(data[:nullIndex])
	}
}

func parseMacAddress(data *GUID) string {
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", data.Data4[2], data.Data4[3], data.Data4[4], data.Data4[5], data.Data4[6], data.Data4[7])
}

func parseDateTimeOffset(g *GUID) time.Time {
	// The 60-bit timestamp is stored across Data1, Data2, and Data3.
	// Data1: time_low (32 bits)
	// Data2: time_mid (16 bits)
	// Data3: time_hi_and_version (16 bits)

	// Extract the components into 64-bit types
	timeLow := uint64(g.Data1)
	timeMid := uint64(g.Data2)
	timeHiAndVersion := uint64(g.Data3)

	// The version is in the 4 highest bits of Data3.
	// We must mask it out to get the 12-bit time_hi component.
	// (e.g., 0x11D1 & 0x0FFF = 0x01D1)
	timeHi := timeHiAndVersion & 0x0FFF

	// Re-assemble the 60-bit timestamp
	// (time_hi << 48) | (time_mid << 32) | time_low
	timestamp100ns := int64((timeHi << 48) | (timeMid << 32) | timeLow)

	// 'uuidEpochOffset' is the number of 100-ns intervals
	// between the UUID epoch (Oct 15, 1582) and the Go/Unix epoch (Jan 1, 1970).
	const uuidEpochOffset = 122192928000000000

	// Get the 100-ns intervals since the Go epoch
	goTimestamp100ns := timestamp100ns - uuidEpochOffset

	// Convert from 100-ns intervals to nanoseconds
	goTimestampNanos := goTimestamp100ns * 100

	// Create the time.Time object in UTC
	return time.Unix(0, goTimestampNanos).UTC()
}

func NewDestListEntry(data []byte, version int32) (*DestListEntry, error) {

	var interactionCount int32
	var unknown3 int32
	var unknown4 int32
	var path string

	// Version 1 has less fields than later versions
	if version > 1 {
		if len(data) < 130 {
			return nil, fmt.Errorf("data is too short to contain a version %d DestListEntry", version)
		}

		pathLength := int(binary.LittleEndian.Uint16(data[128:130])) * 2
		if len(data) < 130+pathLength {
			return nil, fmt.Errorf("data is too short to contain a version %d DestListEntry", version)
		}

		interactionCount = int32(binary.LittleEndian.Uint32(data[116:120]))
		unknown3 = int32(binary.LittleEndian.Uint32(data[120:124]))
		unknown4 = int32(binary.LittleEndian.Uint32(data[124:128]))
		path = string(data[130 : 130+pathLength])
	} else {
		if len(data) < 114 {
			return nil, fmt.Errorf("data is too short to contain a version %d DestListEntry", version)
		}
		pathLength := int(binary.LittleEndian.Uint16(data[112:114]) * 2)
		if len(data) < 114+pathLength {
			return nil, fmt.Errorf("data is too short to contain a version %d DestListEntry", version)
		}
		path = string(data[114 : 114+pathLength])
	}

	checksum := binary.LittleEndian.Uint64(data[0:8])
	volumeDroid := NewGUID(data[8:24])
	fileDroid := NewGUID(data[24:40])
	volumeBirthDroid := NewGUID(data[40:56])
	fileBirthDroid := NewGUID(data[56:72])
	hostname := parseHostname(data[72:88])
	entryNumber := int32(binary.LittleEndian.Uint32(data[88:92]))
	streamName := fmt.Sprintf("%x", entryNumber)
	unknown0 := int32(binary.LittleEndian.Uint32(data[92:96]))
	accessCount := float32(binary.LittleEndian.Uint32(data[96:100]))
	lastModifiedTime := toTime(data[100:108])
	pinStatus := int32(binary.LittleEndian.Uint32(data[108:112]))
	macAddress := parseMacAddress(fileDroid)
	creationTime := parseDateTimeOffset(fileDroid)

	return &DestListEntry{
		Checksum:         checksum,
		VolumeDroid:      volumeDroid,
		FileDroid:        fileDroid,
		VolumeBirthDroid: volumeBirthDroid,
		FileBirthDroid:   fileBirthDroid,
		Hostname:         hostname,
		EntryNumber:      entryNumber,
		Unknown0:         unknown0,
		AccessCount:      accessCount,
		LastModifiedTime: lastModifiedTime,
		PinStatus:        pinStatus,
		InteractionCount: interactionCount,
		Unknown3:         unknown3,
		Unknown4:         unknown4,
		Path:             path,
		MacAddress:       macAddress,
		CreationTime:     creationTime,
		StreamName:       streamName,
	}, nil
}

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

func (e *DestListEntry) String() string {
	sb := strings.Builder{}
	sb.WriteString("<Entry: {")
	sb.WriteString(fmt.Sprintf("Checksum: %d, ", e.Checksum))
	sb.WriteString(fmt.Sprintf("VolumeDroid: %s, ", e.VolumeDroid))
	sb.WriteString(fmt.Sprintf("FileDroid: %s, ", e.FileDroid))
	sb.WriteString(fmt.Sprintf("VolumeBirthDroid: %s, ", e.VolumeBirthDroid))
	sb.WriteString(fmt.Sprintf("FileBirthDroid: %s, ", e.FileBirthDroid))
	sb.WriteString(fmt.Sprintf("Hostname: %s, ", e.Hostname))
	sb.WriteString(fmt.Sprintf("EntryNumber: %d, ", e.EntryNumber))
	sb.WriteString(fmt.Sprintf("Unknown0: %d, ", e.Unknown0))
	sb.WriteString(fmt.Sprintf("AccessCount: %f, ", e.AccessCount))
	sb.WriteString(fmt.Sprintf("LastModifiedTime: %s, ", e.LastModifiedTime))
	sb.WriteString(fmt.Sprintf("PinStatus: %d, ", e.PinStatus))
	sb.WriteString(fmt.Sprintf("InteractionCount: %d, ", e.InteractionCount))
	sb.WriteString(fmt.Sprintf("Unknown3: %d, ", e.Unknown3))
	sb.WriteString(fmt.Sprintf("Unknown4: %d, ", e.Unknown4))
	sb.WriteString(fmt.Sprintf("Path: %s, ", e.Path))
	sb.WriteString(fmt.Sprintf("MacAddress: %s, ", e.MacAddress))
	sb.WriteString(fmt.Sprintf("CreationTime: %s", e.CreationTime))
	sb.WriteString("}")
	return sb.String()
}

// Olecfb represents a microsoft compound file binary file.
// tailored for jump list files.
// - path: the path to the file
// - streams: a map of stream names to their data
type Olecfb struct {
	Path           string
	DestList       *DestList
	Lnks           []*Lnk
	UnknownStreams map[string][]byte
}

func (o *Olecfb) HasValidDestList() bool {
	return o.DestList != nil
}

// NewOlecfb creates a new Olecfb object.
// - path: the path to the file
// - log: the logger to use
// returns: a new Olecfb object, or an error if the file cannot be opened or parsed
func NewOlecfb(path string, log *logger.Logger) (*Olecfb, error) {
	olecfb := &Olecfb{
		Path:           path,
		DestList:       nil,
		Lnks:           make([]*Lnk, 0),
		UnknownStreams: make(map[string][]byte),
	}
	
	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Parse the file as a Microsoft Compound File Binary (OLECFB)
	doc, err := mscfb.New(file)
	if err != nil {
		return nil, err
	}

	streams := make(map[string][]byte)
	// Iterate over the entries in the OLECFB
	for entry, err := doc.Next(); err == nil; entry, err = doc.Next() {
		streamName := strings.ToLower(entry.Name)
		streams[streamName], err = io.ReadAll(entry)
		if err != nil {
			return nil, fmt.Errorf("failed to read stream: %w", err)
		}
	}

	// Parse the DestList stream
	destList, err := NewDestList(streams[strings.ToLower(DestListStreamName)], log)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DestList: %w", err)
	}
	olecfb.DestList = destList

	for name, data := range streams {
		fmt.Println("Stream: ", name, "Length: ", len(data))
	}

	for _, entry := range destList.Entries {
		_, ok := streams[entry.StreamName]; if !ok {
			fmt.Println("Stream not found: ", entry.StreamName)
			continue
		}
		isLnk := IsLnkSignature(streams[entry.StreamName]) 
		fmt.Printf("Entry: %s Number: %d streamExists: %t isLnk: %t\n", entry.StreamName, entry.EntryNumber, ok, isLnk)
	}

	// // If the entry is a LNK file, parse it.
	// if IsLnkSignature(data) {
	// 	lnk, err := NewLnkFromBytes(data, 0, log)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to parse LNK file: %w", err)
	// 	}
	// 	olecfb.Lnks = append(olecfb.Lnks, lnk)
	// 	continue
	// }
	// // If the entry is not a DestList stream or a LNK file, store the data in the UnknownStreams map.
	// olecfb.UnknownStreams[entry.Name] = data
	return olecfb, nil
}
