// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package shell_items

import (
	"encoding/binary"
	"fmt"
	"time"
)

// ExtractDateTimeOffsetFromBytes parses a 4-byte MS-DOS timestamp into a Go Time pointer.
// It returns nil if the slice is too short or the date components are invalid.
//
// The format assumes:
// Bytes 0-1: Date (Bits 0-4: Day, 5-8: Month, 9-15: Year offset from 1980)
// Bytes 2-3: Time (Bits 11-15: Hour, 5-10: Minute, 0-4: Second/2)
// This function was written originally by gemini-2.5, edited by myself to fit the needs of this project.
// The original function is located at:
// https://github.com/EricZimmerman/ExtensionBlocks/blob/58e35b8457bf3006f672c972619bc0fb913fb7e4/ExtensionBlocks/Utils.cs#L4222-L4259
func extractDateTimeOffsetFromBytes(rawBytes []byte) (*time.Time, error) {
	// 1. Safety Check: Ensure we have at least 4 bytes to avoid panics
	if len(rawBytes) < 4 {
		return nil, fmt.Errorf("error extracting date time: raw bytes is too short")
	}

	datePart := binary.LittleEndian.Uint16(rawBytes[0:2])
	// Extract Day (Bits 0-4)
	day := int(datePart & 0x1f)
	// Extract Month (Bits 5-8)
	// We mask 0x1e0 (binary 111100000) and shift right by 5
	month := int((datePart & 0x1e0) >> 5)

	// Extract Year (Bits 9-15)
	// We mask 0xfe00 and shift right by 9, then add the 1980 epoch
	year := int((datePart&0xfe00)>>9) + 1980

	// 3. Parse the Time (Next 2 bytes)
	timePart := binary.LittleEndian.Uint16(rawBytes[2:4])

	// Extract Hour (Top 5 bits: 11-15)
	hour := int((timePart >> 11) & 0x1f)

	// Extract Minute (Middle 6 bits: 5-10)
	minute := int((timePart >> 5) & 0x3f)

	// Extract Seconds (Bottom 5 bits: 0-4)
	// DOS stores seconds divided by 2, so we multiply by 2.
	seconds := int(timePart&0x1f) * 2

	// 4. Validation to ensure the date and time are valid
	if month < 1 || month > 12 || day < 1 || day > 31 || hour > 23 || minute > 59 || seconds > 60 {
		return nil, fmt.Errorf("error extracting date time: invalid date or time")
	}

	// 5. Construct the final time in UTC
	dt := time.Date(year, time.Month(month), day, hour, minute, seconds, 0, time.UTC)

	return &dt, nil
}