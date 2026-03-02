// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"

	golnk "github.com/parsiya/golnk"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

// LnkSignature is the signature for a LNK file.
var LnkSignature = []byte{0x4c, 0x00, 0x00, 0x00}

// MinLnkSize is the minimum size of a LNK file, anything smaller can not be a valid LNK file
// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/Lnk.cs#L24-L28
var MinLnkSize = 76

// Lnk is a struct that contains the data for a LNK file.
// It is used to store the data for a LNK file.
// There is more data in the LNK file, but we are only interested in the data that is relevant forensically
// I used the github.com/EricZimmerman/Lnk library for reference when creating this struct, and the fields
// are similar to the output of jlecmd which leverages the same library.
type Lnk struct {
	LocalPath              string    `osquery:"local_path"`
	FileSize               uint32    `osquery:"file_size"`
	HotKey                 string    `osquery:"hot_key"`
	IconIndex              int32     `osquery:"icon_index"`
	ShowWindow             string    `osquery:"show_window"`
	IconLocation           string    `osquery:"icon_location"`
	CommandLineArguments   string    `osquery:"command_line_arguments"`
	TargetModificationDate time.Time `osquery:"target_modification_time"`
	TargetLastAccessedDate time.Time `osquery:"target_last_accessed_time"`
	TargetCreationDate     time.Time `osquery:"target_creation_time"`
	VolumeSerialNumber     string    `osquery:"volume_serial_number"`
	VolumeType             string    `osquery:"volume_type"`
	VolumeLabel            string    `osquery:"volume_label"`
	VolumeLabelOffset      uint32    `osquery:"volume_label_offset"`
	Name                   string    `osquery:"name"`
}

// newLnkFromBytes creates a new Lnk object from a byte slice.
// It is used to create a new Lnk object from a byte slice.
func newLnkFromBytes(data []byte, log *logger.Logger) (*Lnk, error) {
	if len(data) < MinLnkSize {
		return nil, fmt.Errorf("data is too short to contain a valid LNK file")
	}

	// Check if the data contains a LNK signature
	if !bytes.Equal(data[:len(LnkSignature)], LnkSignature) {
		return nil, fmt.Errorf("not a LNK file")
	}

	// Read the LNK file using the golnk library
	// github.com/parsiya/golnk
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

	// the golnk library returns "No Key Assigned" if no hotkey is set,
	// we need to convert this to an empty string
	hotKey := lnkFile.Header.HotKey
	if hotKey == "No Key Assigned" {
		hotKey = ""
	}

	lnk := &Lnk{
		LocalPath:              lnkFile.LinkInfo.LocalBasePath,
		FileSize:               lnkFile.Header.TargetFileSize,
		HotKey:                 hotKey,
		IconIndex:              lnkFile.Header.IconIndex,
		ShowWindow:             lnkFile.Header.ShowCommand,
		IconLocation:           lnkFile.StringData.IconLocation,
		TargetModificationDate: lnkFile.Header.WriteTime,
		TargetLastAccessedDate: lnkFile.Header.AccessTime,
		TargetCreationDate:     lnkFile.Header.CreationTime,
		VolumeSerialNumber:     volumeSerialNumber,
		VolumeType:             lnkFile.LinkInfo.VolID.DriveType,
		VolumeLabel:            lnkFile.LinkInfo.VolID.VolumeLabel,
		VolumeLabelOffset:      lnkFile.LinkInfo.VolID.VolumeLabelOffset,
		CommandLineArguments:   lnkFile.StringData.CommandLineArguments,
		Name:                   lnkFile.StringData.NameString,
	}

	return lnk, nil
}
