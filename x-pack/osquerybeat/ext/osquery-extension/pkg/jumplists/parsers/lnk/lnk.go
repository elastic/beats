// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package lnk

import (
	"bytes"
	"fmt"
	"os"

	golnk "github.com/parsiya/golnk"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/parsers/lnk/shell_items"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

var LnkSignature = []byte{0x4c, 0x00, 0x00, 0x00}
var LnkFooterSignature = []byte{0xAB, 0xFB, 0xBF, 0xBA}

// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/Lnk.cs#L24-L28
var MinLnkSize = 76

type Lnk struct {
	golnk.LnkFile
	ShellItems []shell_items.ShellItem
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

	var shellItems []shell_items.ShellItem
	if lnkFile.Header.LinkFlags["HasLinkTargetIDList"] {
		for _, item := range lnkFile.IDList.List.ItemIDList {
			shellItems = append(shellItems, shell_items.NewShellItem(item.Size, item.Data))
		}
	}

	return &Lnk{LnkFile: lnkFile, ShellItems: shellItems}, nil
}
