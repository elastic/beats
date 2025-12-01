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
		TargetModifiedTime:   lnkFile.Header.WriteTime,
		TargetAccessedTime:   lnkFile.Header.AccessTime,
		TargetCreatedTime:    lnkFile.Header.CreationTime,
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
