// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	golnk "github.com/parsiya/golnk"

	//"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/parsers/lnk/shell_items"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

var LnkSignature = []byte{0x4c, 0x00, 0x00, 0x00}

// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/Lnk.cs#L24-L28
var MinLnkSize = 76

type Lnk struct {
	EntryNumber          int       `osquery:"entry_number"`
	TargetPath           string    `osquery:"target_path"`
	IconLocation         string    `osquery:"icon_location"`
	CommandLineArguments string    `osquery:"command_line_arguments"`
	TargetModifiedTime   time.Time `osquery:"target_modified_time" format:"unix"`
	TargetAccessedTime   time.Time `osquery:"target_accessed_time" format:"unix"`
	TargetCreatedTime    time.Time `osquery:"target_created_time" format:"unix"`
	VolumeSerialNumber   string    `osquery:"volume_serial_number"`
	VolumeType           string    `osquery:"volume_type"`
	VolumeLabel          string    `osquery:"volume_label"`
	WorkingDir           string    `osquery:"working_dir"`
	NameString           string    `osquery:"name_string"`
	RelativePath         string    `osquery:"relative_path"`
}

func NewLnkFromPath(filePath string, log *logger.Logger) (*Lnk, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read LNK file: %w", err)
	}
	return NewLnkFromBytes(bytes, 0, log)
}

func toTime(t []byte) time.Time {
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

func NewLnkFromBytes(data []byte, entryNumber int, log *logger.Logger) (*Lnk, error) {
	if len(data) < len(LnkSignature) {
		return nil, fmt.Errorf("data is too short to contain a LNK signature")
	}

	if !bytes.Equal(data[:len(LnkSignature)], LnkSignature) {
		return nil, fmt.Errorf("not a LNK file")
	}

	if len(data) < MinLnkSize {
		return nil, fmt.Errorf("data is too short to contain a valid LNK file")
	}

	// There appears to be a bug in the golnk library, where the access time, creation time, and modification time
	// are not being converted to the correct time.Time values in cases where the time is zero (not set).
	// I have submitted an issue to the golnk library: https://github.com/parsiya/golnk/issues/7
	// as well as a pull request to fix it: https://github.com/parsiya/golnk/pull/8
	// In the meantime, we will convert the timestamps manually
	// offset values pulled from https://github.com/EricZimmerman/Lnk/blob/master/Lnk/Header.cs#L134-L146
	accessTime := toTime(data[28:36])
	creationTime := toTime(data[36:44])
	modificationTime := toTime(data[44:52])

	lnkFile, err := golnk.Read(bytes.NewReader(data), uint64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to read LNK file: %w", err)
	}

	// the golnk library returns the drive serial number as a hex string representation of a uint32
	// we need to convert it to a string in the format of XXXX-XXXX that windows displays
	var volumeSerialNumber string
	cleanInput := strings.TrimPrefix(lnkFile.LinkInfo.VolID.DriveSerialNumber, "0x")
	val, err := strconv.ParseUint(cleanInput, 16, 32)
	if err == nil {
		bytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(bytes, uint32(val))
		volumeSerialNumber = fmt.Sprintf("%02X%02X-%02X%02X", bytes[0], bytes[1], bytes[2], bytes[3])
	}

	lnk := &Lnk{
		EntryNumber:          entryNumber,
		TargetPath:           lnkFile.LinkInfo.LocalBasePath,
		IconLocation:         lnkFile.StringData.IconLocation,
		TargetModifiedTime:   modificationTime,
		TargetAccessedTime:   accessTime,
		TargetCreatedTime:    creationTime,
		VolumeSerialNumber:   volumeSerialNumber,
		VolumeType:           lnkFile.LinkInfo.VolID.DriveType,
		VolumeLabel:          lnkFile.LinkInfo.VolID.VolumeLabel,
		CommandLineArguments: lnkFile.StringData.CommandLineArguments,
		WorkingDir:           lnkFile.StringData.WorkingDir,
		NameString:           lnkFile.StringData.NameString,
		RelativePath:         lnkFile.StringData.RelativePath,
	}

	return lnk, nil
}
