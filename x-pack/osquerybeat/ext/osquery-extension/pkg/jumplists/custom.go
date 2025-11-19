// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"bytes"
	"fmt"
	"os"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/parsers/lnk"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/parsers/resources"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

type CustomJumpList struct {
	appId resources.ApplicationId
	path  string
	lnks  []*lnk.Lnk
}

func NewCustomJumpList(filePath string, log *logger.Logger) (*CustomJumpList, error) {
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	lnks, err := carveLnkFiles(fileBytes, log)
	if err != nil {
		return nil, fmt.Errorf("failed to carve LNK files: %w", err)
	}
	return &CustomJumpList{
		appId: resources.GetAppIdFromFileName(filePath, log),
		path:  filePath,
		lnks:  lnks,
	}, nil
}

func (c *CustomJumpList) Path() string {
	return c.path
}

func (c *CustomJumpList) AppId() resources.ApplicationId {
	return c.appId
}

func (c *CustomJumpList) Type() JumpListType {
	return JumpListTypeCustom
}

func GetCustomJumpLists(log *logger.Logger) ([]*CustomJumpList, error) {
	files, err := FindJumplistFiles(JumpListTypeCustom, log)
	if err != nil {
		return nil, err
	}
	var jumpLists []*CustomJumpList
	for _, file := range files {
		jumpList, err := NewCustomJumpList(file, log)
		if err != nil {
			log.Errorf("failed to parse Custom Jump List: %v", err)
			continue
		}
		jumpLists = append(jumpLists, jumpList)
	}
	return jumpLists, nil
}

func carveLnkFiles(fileBytes []byte, log *logger.Logger) ([]*lnk.Lnk, error) {
	// A custom destination file contains one or more LNK files.
	// We need to scan the file looking for LNK signatures and carve out the individual LNK files.

	var lnks []*lnk.Lnk

	// Scan through file looking for LNK signatures
	for i := 0; i < len(fileBytes); i++ {

		// Check if we found a LNK signature
		if len(fileBytes[i:]) < len(lnk.LnkSignature) {
			// stop scanning if we encounter a short buffer
			break
		}

		signatureSlice := fileBytes[i : i+len(lnk.LnkSignature)]
		if !bytes.Equal(signatureSlice, lnk.LnkSignature) {
			continue
		}

		// Found a LNK signature, so we can carve out the file
		// Find end - either next signature or EOF
		start := i
		end := len(fileBytes)

		searchStart := start + len(lnk.LnkSignature) // skip the signature we just found
		searchEnd := len(fileBytes) - len(lnk.LnkSignature) // stop at the next signature or EOF

		for j := searchStart; j < searchEnd; j++ {
			nextSignature := fileBytes[j : j+len(lnk.LnkSignature)]
			if bytes.Equal(nextSignature, lnk.LnkSignature) {
				end = j
				break
			}
		}

		os.WriteFile(fmt.Sprintf("lnk_%d.bin", i), fileBytes[start:end], 0644)

		// Carve out the LNK file, and convert it to an Lnk
		lnkFile, err := lnk.NewLnkFromBytes(fileBytes[start:end], log)
		if err != nil {
			return nil, fmt.Errorf("failed to read LNK file: %w", err)
		}
		lnks = append(lnks, lnkFile)

		i = end - 1 // Move cursor to end (minus 1 since loop will increment)
	}
	return lnks, nil
}
