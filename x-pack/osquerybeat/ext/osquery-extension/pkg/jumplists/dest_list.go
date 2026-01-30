// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

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
	mruPosition      int32
	checksum         uint64
	volumeDroid      *GUID
	fileDroid        *GUID
	volumeBirthDroid *GUID
	fileBirthDroid   *GUID
	Hostname         string `osquery:"hostname"`
	EntryNumber      int32  `osquery:"entry_number"`
	unknown0         int32
	AccessCount      float32   `osquery:"access_count"`
	LastModifiedTime time.Time `osquery:"last_modified_time"`
	PinStatus        bool      `osquery:"is_pinned"`
	InteractionCount int32     `osquery:"interaction_count"`
	unknown3         int32
	unknown4         int32
	Path             string    `osquery:"dest_entry_path"`
	ResolvedPath     string    `osquery:"dest_entry_path_resolved"`
	MacAddress       string    `osquery:"mac_address"`
	CreationTime     time.Time `osquery:"creation_time"`
	name             string
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
			entry, err := NewDestListEntry(entryBytes, header.Version, log)
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
			entry, err := NewDestListEntry(entryBytes, header.Version, log)
			if err != nil {
				return nil, fmt.Errorf("failed to parse DestList entry: %w", err)
			}
			destList.Entries = append(destList.Entries, entry)
			index = entryEnd
		}
	}
	return destList, nil
}

func readInt32(b []byte) int32 {
	val := binary.LittleEndian.Uint32(b)
	if val > math.MaxInt32 {
		return -1
	}
	return int32(val)
}

func readFloat32(b []byte) float32 {
	bits := binary.LittleEndian.Uint32(b)
	return math.Float32frombits(bits)
}

func NewDestListHeader(data []byte) (*DestListHeader, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("data is too short to contain a DestListHeader")
	}

	header := &DestListHeader{
		Version:               readInt32(data[0:4]),
		NumberOfEntries:       readInt32(data[4:8]),
		NumberOfPinnedEntries: readInt32(data[8:12]),
		UnknownCounter:        readFloat32(data[12:16]),
		LastEntryNumber:       readInt32(data[16:20]),
		Unknown1:              readInt32(data[20:24]),
		LastRevisionNumber:    readInt32(data[24:28]),
		Unknown2:              readInt32(data[28:32]),
	}
	return header, nil
}

func parseHostname(data []byte) string {
	if len(data) < 16 {
		return ""
	}

	// The hostname can be either a UTF-16 encoded string or a null-terminated string.
	// If the first byte is 0, then it is a UTF-16 encoded string.
	// If the first byte is not 0, then it is a null-terminated string.
	// In either case, read the string until the first null byte.
	var hostname string
	if data[1] == 0 {
		utf16data := make([]uint16, 0, 8)
		for i := range 8 {
			start := i * 2
			end := start + 2
			val := binary.LittleEndian.Uint16(data[start:end])
			if val == 0 {
				break
			}
			utf16data = append(utf16data, val)
		}
		hostname = string(utf16.Decode(utf16data))
	} else {
		size := 16
		if idx := bytes.IndexByte(data[:16], 0); idx >= 0 {
			size = idx
		}
		hostname = string(data[:size])
	}
	return hostname
}

// Control panel categories are stored as integers in the path.
// This function converts the integer to a string using the known values.
// https://github.com/EricZimmerman/RegistryPlugins/blob/0f70778fb1481aff9b4deada524cc68bf1367b56/RegistryPlugin.LastVisitedPidlMRU/ShellItems/ShellBag0x01.cs#L39-L109
func resolveControlPanelCategory(part string) string {
	partInt, err := strconv.Atoi(part)
	if err != nil {
		return part
	}
	switch partInt {
	case 0x00:
		return "All Control Panel Items"
	case 0x01:
		return "Appearance and Personalization"
	case 0x02:
		return "Hardware and Sound"
	case 0x03:
		return "Network and Internet"
	case 0x04:
		return "Sound, Speech and Audio Devices"
	case 0x05:
		return "System and Security"
	case 0x06:
		return "Clock, Language, and Region"
	case 0x07:
		return "Ease of Access"
	case 0x08:
		return "Programs"
	case 0x09:
		return "User Accounts"
	case 0x10:
		return "Security Center"
	case 0x11:
		return "Mobile PC"
	default:
		return fmt.Sprintf("Unknown category! Category ID: %d", partInt)
	}
}

func resolvePath(path string) string {
	// Known folders start with "knownfolder" and are followed by a GUID.
	if len(path) >= 50 && strings.HasPrefix(path, "knownfolder") {
		knownFolderGuid := path[13:49]
		knownFolder, ok := LookupGuidMapping(knownFolderGuid)
		if ok {
			return knownFolder
		}
	}

	// If the path does not contain ::{, then it is not a constructed path.
	if !strings.Contains(path, "::{") {
		return ""
	}

	// If the path contains ::{, then it is a constructed path with GUIDs.
	// Split the path into parts and iterate over them.
	parts := strings.Split(path, "\\")
	sb := strings.Builder{}
	previousResolvedPart := ""

	// Iterate over the parts and translate the GUIDs to strings
	for i, part := range parts {
		if i > 0 {
			sb.WriteString("\\")
		}

		resolvedPart := part

		// If the part starts with ::{, then it is a GUID.
		if strings.HasPrefix(part, "::{") && len(part) == 40 {
			guidString := part[3:39]
			knownFolder, ok := LookupGuidMapping(guidString)
			if ok {
				resolvedPart = knownFolder
			}
		} else {
			// The Control Panel GUID can be followed by an integer that represents the control panel category.
			if i > 0 && previousResolvedPart == "ControlPanelHome" {
				resolvedPart = resolveControlPanelCategory(part)
			}
		}
		// Add the resolved part to the string builder.
		sb.WriteString(resolvedPart)
		// Update the previous resolved part.
		previousResolvedPart = resolvedPart
	}

	// Return the resolved path.
	return sb.String()
}

func parseTimestamp(t []byte) time.Time {
	if len(t) != 8 {
		return time.Time{}
	}

	// read the low 32 bits and the high 32 bits as uint32 (little endian)
	dwLow := binary.LittleEndian.Uint32(t[4:])
	dwHigh := binary.LittleEndian.Uint32(t[:4])

	// combine the low and high 32 bits into a single 64 bit integer
	// this is the number of 100 nanosecond intervals since January 1, 1601 (UTC)
	ticks := int64(dwLow)<<32 + int64(dwHigh)

	// if the ticks are less than the number of 100 nanosecond intervals since January 1, 1601 (UTC), the time is invalid
	// so return zero time
	if ticks < 116444736000000000 {
		return time.Time{}
	}

	// subtract the number of 100 nanosecond representing the unix epoch (January 1, 1970 (UTC))
	ticks -= 116444736000000000

	// convert the ticks to seconds and nanoseconds
	// the ticks are in 100 nanosecond intervals, so we need to divide by 10000000 to get seconds
	// and take the remainder to get nanoseconds
	seconds := ticks / 10000000
	nanos := (ticks % 10000000) * 100

	// return the time as a time.Time value
	return time.Unix(seconds, nanos)
}

func NewDestListEntry(data []byte, version int32, log *logger.Logger) (*DestListEntry, error) {
	var interactionCount int32
	var unknown3 int32
	var unknown4 int32
	var rawPath []uint16

	// Version 1 has less fields than later versions
	if version > 1 {
		if len(data) < 130 {
			return nil, fmt.Errorf("data is too short to contain a version %d DestListEntry", version)
		}

		interactionCount = readInt32(data[116:120])
		unknown3 = readInt32(data[120:124])
		unknown4 = readInt32(data[124:128])

		pathLength := int(binary.LittleEndian.Uint16(data[128:130]))
		if len(data) < 130+(pathLength*2) {
			return nil, fmt.Errorf("data is too short to contain a version %d DestListEntry", version)
		}
		u16s := make([]uint16, pathLength)
		for i := range pathLength {
			offset := 130 + (i * 2)
			u16s[i] = binary.LittleEndian.Uint16(data[offset : offset+2])
		}
		rawPath = u16s

	} else {
		if len(data) < 114 {
			return nil, fmt.Errorf("data is too short to contain a version %d DestListEntry", version)
		}
		pathLength := int(binary.LittleEndian.Uint16(data[112:114]) * 2)
		if len(data) < 114+pathLength {
			return nil, fmt.Errorf("data is too short to contain a version %d DestListEntry", version)
		}
		u16s := make([]uint16, pathLength)
		for i := range pathLength {
			offset := 130 + (i * 2)
			u16s[i] = binary.LittleEndian.Uint16(data[offset : offset+2])
		}
		rawPath = u16s
	}

	checksum := binary.LittleEndian.Uint64(data[0:8])
	volumeDroid := NewGUID(data[8:24])
	fileDroid := NewGUID(data[24:40])
	volumeBirthDroid := NewGUID(data[40:56])
	fileBirthDroid := NewGUID(data[56:72])
	hostname := parseHostname(data[72:88])
	entryNumber := readInt32(data[88:92])
	name := fmt.Sprintf("%x", entryNumber)
	unknown0 := readInt32(data[92:96])
	accessCount := readFloat32(data[96:100])
	lastModifiedTime := parseTimestamp(data[100:108])
	pinStatus := readInt32(data[108:112])
	macAddress := fileDroid.AsMacAddress()
	creationTime := fileDroid.AsFileTime()
	path := string(utf16.Decode(rawPath))
	resolvedPath := resolvePath(path)

	return &DestListEntry{
		checksum:         checksum,
		volumeDroid:      volumeDroid,
		fileDroid:        fileDroid,
		volumeBirthDroid: volumeBirthDroid,
		fileBirthDroid:   fileBirthDroid,
		Hostname:         hostname,
		EntryNumber:      entryNumber,
		unknown0:         unknown0,
		AccessCount:      accessCount,
		LastModifiedTime: lastModifiedTime,
		PinStatus:        pinStatus != -1,
		InteractionCount: interactionCount,
		unknown3:         unknown3,
		unknown4:         unknown4,
		Path:             path,
		ResolvedPath:     resolvedPath,
		MacAddress:       macAddress,
		CreationTime:     creationTime,
		name:             name,
	}, nil
}
