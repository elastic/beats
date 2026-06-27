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

	"github.com/Microsoft/go-winio/pkg/guid"

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
	checksum         uint64
	volumeDroid      string
	fileDroid        string
	volumeBirthDroid string
	fileBirthDroid   string
	Hostname         string `osquery:"hostname"`
	EntryNumber      int32  `osquery:"entry_number"`
	unknown0         int32
	AccessCount      float32
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

	// Version 1 stores the path size at offset 112; later versions at offset 128.
	pathSizeOffset := 128
	if header.Version == 1 {
		pathSizeOffset = 112
	}

	index := DestListHeaderSize
	for i := range int(header.NumberOfEntries) {
		entryStart := index

		// Read the path size (uint16) at the version-dependent offset.
		pathSizeEnd := index + pathSizeOffset + 2
		if pathSizeEnd > len(data) {
			return nil, fmt.Errorf("not enough data to read path size for entry %d", i)
		}
		pathSize := int(binary.LittleEndian.Uint16(data[index+pathSizeOffset : pathSizeEnd]))

		// The entry extends past the path size field by pathSize UTF-16 code units.
		entryEnd := pathSizeEnd + (pathSize * 2)

		// Version 2+ entries have an additional Serialized Property Store after the path.
		if header.Version > 1 {
			if entryEnd+4 > len(data) {
				return nil, fmt.Errorf("not enough data to read sps size for entry %d", i)
			}
			spsSize := int(binary.LittleEndian.Uint32(data[entryEnd : entryEnd+4]))
			entryEnd += 4 + spsSize
		}

		if entryEnd > len(data) {
			return nil, fmt.Errorf("not enough data to read entry %d", i)
		}

		entry, err := NewDestListEntry(data[entryStart:entryEnd], header.Version, log)
		if err != nil {
			return nil, fmt.Errorf("failed to parse DestList entry %d: %w", i, err)
		}
		destList.Entries = append(destList.Entries, entry)
		index = entryEnd
	}

	return destList, nil
}

func readInt32(b []byte) int32 {
	if len(b) < 4 {
		return 0
	}
	val := binary.LittleEndian.Uint32(b)
	if val > math.MaxInt32 {
		return -1
	}
	return int32(val)
}

func readFloat32(b []byte) float32 {
	if len(b) < 4 {
		return 0
	}
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

func AsFileTime(g guid.GUID) time.Time {
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
	// Use uint64 for the calculation to avoid G115 overflow warnings
	uVal := (timeHi << 48) | (timeMid << 32) | timeLow
	timestamp100ns := int64(0)
	if uVal < math.MaxInt64 {
		timestamp100ns = int64(uVal)
	}

	// 'uuidEpochOffset' is the number of 100-ns intervals
	// between the UUID epoch (Oct 15, 1582) and the Go/Unix epoch (Jan 1, 1970).
	const uuidEpochOffset = 122192928000000000

	// Get the 100-ns intervals since the Go epoch
	unixTimestamp100ns := timestamp100ns - uuidEpochOffset

	// We divide by 10,000,000 to get seconds (10 million 100ns ticks per second)
	seconds := unixTimestamp100ns / 10_000_000
	// We take the remainder and multiply by 100 to get nanoseconds
	nanos := (unixTimestamp100ns % 10_000_000) * 100

	// Create the time.Time object in UTC
	return time.Unix(seconds, nanos).UTC()
}

func AsMacAddress(g guid.GUID) string {
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", g.Data4[2], g.Data4[3], g.Data4[4], g.Data4[5], g.Data4[6], g.Data4[7])
}

func LookupGuidMapping(guid string) (string, bool) {
	if knownFolder, ok := knownFolderMappings[guid]; ok {
		return knownFolder, true
	}
	return "", false
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
		pathLength := int(binary.LittleEndian.Uint16(data[112:114]))
		byteLength := pathLength * 2
		if len(data) < 114+byteLength {
			return nil, fmt.Errorf("data is too short to contain a version %d DestListEntry", version)
		}
		u16s := make([]uint16, pathLength)
		for i := range pathLength {
			offset := 114 + (i * 2)
			u16s[i] = binary.LittleEndian.Uint16(data[offset : offset+2])
		}
		rawPath = u16s
	}

	checksum := binary.LittleEndian.Uint64(data[0:8])
	volumeDroid := guid.FromWindowsArray([16]byte(data[8:24]))
	fileDroid := guid.FromWindowsArray([16]byte(data[24:40]))
	volumeBirthDroid := guid.FromWindowsArray([16]byte(data[40:56]))
	fileBirthDroid := guid.FromWindowsArray([16]byte(data[56:72]))
	hostname := parseHostname(data[72:88])
	entryNumber := readInt32(data[88:92])
	name := fmt.Sprintf("%x", entryNumber)
	unknown0 := readInt32(data[92:96])
	accessCount := readFloat32(data[96:100])
	lastModifiedTime := parseTimestamp(data[100:108])
	pinStatus := readInt32(data[108:112])
	macAddress := AsMacAddress(fileDroid)
	creationTime := AsFileTime(fileDroid)
	path := string(utf16.Decode(rawPath))
	resolvedPath := resolvePath(path)

	return &DestListEntry{
		checksum:         checksum,
		volumeDroid:      volumeDroid.String(),
		fileDroid:        fileDroid.String(),
		volumeBirthDroid: volumeBirthDroid.String(),
		fileBirthDroid:   fileBirthDroid.String(),
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
