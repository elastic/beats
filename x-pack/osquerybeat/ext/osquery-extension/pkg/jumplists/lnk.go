// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"bytes"
	"fmt"
	"os"
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

func (l *Lnk) String() string {
	sb := strings.Builder{}
	sb.WriteString("Lnk{")
	sb.WriteString(fmt.Sprintf("target_path: %s, ", l.TargetPath))
	sb.WriteString(fmt.Sprintf("target_modified_time: %s, ", l.TargetModifiedTime.UTC().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("target_accessed_time: %s, ", l.TargetAccessedTime.UTC().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("target_created_time: %s, ", l.TargetCreatedTime.UTC().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("volume_serial_number: %s, ", l.VolumeSerialNumber))
	sb.WriteString(fmt.Sprintf("volume_type: %s, ", l.VolumeType))
	sb.WriteString(fmt.Sprintf("volume_label: %s, ", l.VolumeLabel))
	sb.WriteString(fmt.Sprintf("working_dir: %s, ", l.WorkingDir))
	sb.WriteString(fmt.Sprintf("name_string: %s, ", l.NameString))
	sb.WriteString(fmt.Sprintf("relative_path: %s, ", l.RelativePath))
	sb.WriteString(fmt.Sprintf("command_line_arguments: %s}", l.CommandLineArguments))
	return sb.String()
}

func NewLnkFromPath(filePath string, log *logger.Logger) (*Lnk, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read LNK file: %w", err)
	}
	return NewLnkFromBytes(bytes, log)
}

func NewLnkFromBytes(data []byte, log *logger.Logger) (*Lnk, error) {
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

	lnk := &Lnk{
		TargetPath:           lnkFile.LinkInfo.LocalBasePath,
		IconLocation:         lnkFile.StringData.IconLocation,
		TargetModifiedTime:   lnkFile.Header.WriteTime,
		TargetAccessedTime:   lnkFile.Header.AccessTime,
		TargetCreatedTime:    lnkFile.Header.CreationTime,
		VolumeSerialNumber:   lnkFile.LinkInfo.VolID.DriveSerialNumber,
		VolumeType:           lnkFile.LinkInfo.VolID.DriveType,
		VolumeLabel:          lnkFile.LinkInfo.VolID.VolumeLabel,
		CommandLineArguments: lnkFile.StringData.CommandLineArguments,
		WorkingDir:           lnkFile.StringData.WorkingDir,
		NameString:           lnkFile.StringData.NameString,
		RelativePath:         lnkFile.StringData.RelativePath,
	}

	return lnk, nil
}
