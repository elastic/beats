// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package olecfb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/parsers/resources"
)

type DestListEntry struct {
	MruPosition      int32
	Checksum         int64
	VolumeDroid      *resources.GUID
	FileDroid        *resources.GUID
	VolumeBirthDroid *resources.GUID
	FileBirthDroid   *resources.GUID
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
}

func readInt32(data []byte) (int32, error) {
	if len(data) != 4 {
		return 0, fmt.Errorf("data is too short to contain an int32")
	}
	r := bytes.NewReader(data)
	var value int32
	_ = binary.Read(r, binary.LittleEndian, &value)
	return value, nil
}

func readInt64(data []byte) (int64, error) {
	if len(data) != 8 {
		return 0, fmt.Errorf("data is too short to contain an int64")
	}
	r := bytes.NewReader(data)
	var value int64
	_ = binary.Read(r, binary.LittleEndian, &value)
	return value, nil
}

func readFloat32(data []byte) (float32, error) {
	if len(data) != 4 {
		return 0, fmt.Errorf("data is too short to contain a float32")
	}
	r := bytes.NewReader(data)
	var value float32
	_ = binary.Read(r, binary.LittleEndian, &value)
	return value, nil
}

func parseHostname(data []byte) (string, error) {
	if len(data) != 16 {
		return "", fmt.Errorf("data is too short to contain a hostname")
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
		return string(utf16.Decode(utf16data[:nullIndex])), nil
	} else {
		nullIndex := findNullIndex(data)
		return string(data[:nullIndex]), nil
	}
}

func readFileTime(data []byte) (time.Time, error) {
	if len(data) != 8 {
		return time.Time{}, fmt.Errorf("data is too short to contain a file time")
	}

	// 1. Get the 64-bit FILETIME value (little-endian).
	fileTime, err := readInt64(data)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse file time: %w", err)
	}

	// 2. Convert from Windows FILETIME to Unix time.
	// A FILETIME is the number of 100-nanosecond intervals since Jan 1, 1601.
	// A Unix timestamp is seconds/nanoseconds since Jan 1, 1970.

	// This constant is the offset between the two epochs (1601 vs 1970)
	// in 100-nanosecond intervals.
	const epochOffset = 116444736000000000

	// 3. Adjust the epoch and convert from 100-ns intervals to nanoseconds.
	nsec := (fileTime - epochOffset) * 100

	// 4. Create the Go time.Time object in UTC.
	// time.Unix(0, nsec) creates a time from nanoseconds since the Unix epoch.
	// .UTC() matches the C# .ToUniversalTime().
	lastModified := time.Unix(0, nsec).UTC()
	return lastModified, nil
}

func parseMacAddress(data *resources.GUID) string {
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", data.Data4[2], data.Data4[3], data.Data4[4], data.Data4[5], data.Data4[6], data.Data4[7])
}

func parseDateTimeOffset(g *resources.GUID) time.Time {
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
	checksum, err := readInt64(data[0:8])
	if err != nil {
		return nil, fmt.Errorf("failed to parse checksum: %w", err)
	}

	volumeDroid, err := resources.NewGUID(data[8:24])
	if err != nil {
		return nil, fmt.Errorf("failed to parse volume DROID: %w", err)
	}

	fileDroid, err := resources.NewGUID(data[24:40])
	if err != nil {
		return nil, fmt.Errorf("failed to parse file DROID: %w", err)
	}

	volumeBirthDroid, err := resources.NewGUID(data[40:56])
	if err != nil {
		return nil, fmt.Errorf("failed to parse volume birth DROID: %w", err)
	}

	fileBirthDroid, err := resources.NewGUID(data[56:72])
	if err != nil {
		return nil, fmt.Errorf("failed to parse file birth DROID: %w", err)
	}

	hostname, err := parseHostname(data[72:88])
	if err != nil {
		return nil, fmt.Errorf("failed to parse hostname: %w", err)
	}

	entryNumber, err := readInt32(data[88:92])
	if err != nil {
		return nil, fmt.Errorf("failed to parse entry number: %w", err)
	}

	unknown0, err := readInt32(data[92:96])
	if err != nil {
		return nil, fmt.Errorf("failed to parse unknown0: %w", err)
	}

	accessCount, err := readFloat32(data[96:100])
	if err != nil {
		return nil, fmt.Errorf("failed to parse access count: %w", err)
	}

	lastModifiedTime, err := readFileTime(data[100:108])
	if err != nil {
		return nil, fmt.Errorf("failed to parse last modified time: %w", err)
	}

	pinStatus, err := readInt32(data[108:112])
	if err != nil {
		return nil, fmt.Errorf("failed to parse pin status: %w", err)
	}

	var interactionCount int32
	var unknown3 int32
	var unknown4 int32
	var path string

	// Version 1 has less fields than later versions
	if version > 1 {
		interactionCount, err = readInt32(data[116:120])
		if err != nil {
			return nil, fmt.Errorf("failed to parse interaction count: %w", err)
		}

		unknown3, err = readInt32(data[120:124])
		if err != nil {
			return nil, fmt.Errorf("failed to parse unknown3: %w", err)
		}

		unknown4, err = readInt32(data[124:128])
		if err != nil {
			return nil, fmt.Errorf("failed to parse unknown4: %w", err)
		}

		pathLength := binary.LittleEndian.Uint16(data[128:130]) * 2
		path = string(data[130 : 130+pathLength])
	} else {
		pathLength := binary.LittleEndian.Uint16(data[112:114]) * 2
		path = string(data[114 : 114+pathLength])
	}

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
	}, nil
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
