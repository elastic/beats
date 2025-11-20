// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package lnk

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
	target_path            string
	icon_location          string
	command_line_arguments string
	target_modified_time   time.Time
	target_accessed_time   time.Time
	target_created_time    time.Time
	volume_serial_number   string
	volume_type            string
	volume_label           string
	working_dir            string
	name_string            string
	relative_path          string
}

func (l *Lnk) String() string {
	sb := strings.Builder{}
	sb.WriteString("Lnk{")
	sb.WriteString(fmt.Sprintf("target_path: %s, ", l.target_path))
	sb.WriteString(fmt.Sprintf("target_modified_time: %s, ", l.target_modified_time.UTC().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("target_accessed_time: %s, ", l.target_accessed_time.UTC().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("target_created_time: %s, ", l.target_created_time.UTC().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("volume_serial_number: %s, ", l.volume_serial_number))
	sb.WriteString(fmt.Sprintf("volume_type: %s, ", l.volume_type))
	sb.WriteString(fmt.Sprintf("volume_label: %s}", l.volume_label))
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

	// var shellItems []shell_items.ShellItem
	// if lnkFile.Header.LinkFlags["HasLinkTargetIDList"] {
	// 	for _, item := range lnkFile.IDList.List.ItemIDList {
	// 		fmt.Printf("item: %s\n", hex.Dump(item.Data))
	// 		fmt.Printf("item size: %d\n", item.Size)
	// 		fmt.Printf("item data size: %d\n", len(item.Data))
	// 		shellItems = append(shellItems, shell_items.NewShellItem(item.Size, item.Data, log))
	// 	}
	// }

	fmt.Sprintf("StringData: %s\n", lnkFile.StringData)
	fmt.Printf("StringData CommandLineArguments: %s\n", lnkFile.StringData.CommandLineArguments)
	fmt.Printf("StringData IconLocation: %s\n", lnkFile.StringData.IconLocation)
	fmt.Printf("StringData WorkingDir: %s\n", lnkFile.StringData.WorkingDir)
	fmt.Printf("StringData NameString: %s\n", lnkFile.StringData.NameString)
	fmt.Printf("StringData RelativePath: %s\n", lnkFile.StringData.RelativePath)

	lnk := &Lnk{
		target_path:            lnkFile.LinkInfo.LocalBasePath,
		icon_location:          lnkFile.StringData.IconLocation,
		target_modified_time:   lnkFile.Header.WriteTime,
		target_accessed_time:   lnkFile.Header.AccessTime,
		target_created_time:    lnkFile.Header.CreationTime,
		volume_serial_number:   lnkFile.LinkInfo.VolID.DriveSerialNumber,
		volume_type:            lnkFile.LinkInfo.VolID.DriveType,
		volume_label:           lnkFile.LinkInfo.VolID.VolumeLabel,
		command_line_arguments: lnkFile.StringData.CommandLineArguments,
		working_dir:            lnkFile.StringData.WorkingDir,
		name_string:            lnkFile.StringData.NameString,
		relative_path:          lnkFile.StringData.RelativePath,
	}

	return lnk, nil
}
