// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package lnk

import (
	"bytes"
	"os"
	"fmt"

	golnk "github.com/parsiya/golnk"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

func IsLnkSignature(data []byte) bool {
	if len(data) < 4 {
		return false
	}
    lnkSignature := []byte{0x4c, 0x00, 0x00, 0x00}
	return bytes.Equal(data[:4], lnkSignature)
}

type Lnk struct {
	golnk.LnkFile
	ShellItems []*ShellItem
}

func NewLnkFromPath(filePath string, log *logger.Logger) (*Lnk, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read LNK file: %w", err)
	}
	return NewLnkFromBytes(bytes, log)
}

func NewLnkFromBytes(data []byte, log *logger.Logger) (*Lnk, error) {
	if !IsLnkSignature(data) {
		return nil, fmt.Errorf("not a LNK file")
	}
	lnkFile, err := golnk.Read(bytes.NewReader(data), uint64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to read LNK file: %w", err)
	}
	return &Lnk{LnkFile: lnkFile, ShellItems: nil}, nil
}