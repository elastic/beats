// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

type GuidType string

// GUID represents a Windows GUID.
// Data1 is the first 4 bytes of the GUID.
// Data2 is the next 2 bytes of the GUID.
// Data3 is the next 2 bytes of the GUID.
// Data4 is the last 8 bytes of the GUID.
// ExtraData contains extra data about the GUID.
type GUID struct {
	Data1   uint32
	Data2   uint16
	Data3   uint16
	Data4   [8]byte
	Version uint16
}

// String returns the string representation of the GUID.
func (g *GUID) String() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("%08X-%04X-%04X-", g.Data1, g.Data2, g.Data3))
	sb.WriteString(fmt.Sprintf("%02X%02X-", g.Data4[0], g.Data4[1]))
	sb.WriteString(fmt.Sprintf("%02X%02X%02X%02X%02X%02X", g.Data4[2], g.Data4[3], g.Data4[4], g.Data4[5], g.Data4[6], g.Data4[7]))
	return sb.String()
}

// NewGUID creates a new GUID from a 16-byte slice.
func NewGUID(data []byte) *GUID {
	if len(data) < 16 {
		return nil
	}

	data1 := binary.LittleEndian.Uint32(data[:4])
	data2 := binary.LittleEndian.Uint16(data[4:6])
	data3 := binary.LittleEndian.Uint16(data[6:8])
	version := data3 >> 12

	var data4 [8]byte
	copy(data4[:], data[8:16]) // Data4 is big-endian

	guid := &GUID{
		Data1:   data1,
		Data2:   data2,
		Data3:   data3,
		Data4:   data4,
		Version: version,
	}
	return guid
}

func (g *GUID) LookupGuidMapping() (string, bool) {
	return LookupGuidMapping(g.String())
}

func (g *GUID) AsFileTime() time.Time {
	if g.Version != 1 {
		return time.Time{}
	}

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
	uVal := uint64((timeHi << 48) | (timeMid << 32) | timeLow)
	timestamp100ns := int64(uVal)

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

func (g *GUID) AsMacAddress() string {
	if g.Version != 6 {
		return ""
	}
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", g.Data4[2], g.Data4[3], g.Data4[4], g.Data4[5], g.Data4[6], g.Data4[7])
}

func LookupGuidMapping(guid string) (string, bool) {
	if knownFolder, ok := knownFolderMappings[guid]; ok {
		return knownFolder, true
	}
	return "", false
}
