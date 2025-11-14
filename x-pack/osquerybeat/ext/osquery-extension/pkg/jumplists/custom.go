// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"bytes"
	"fmt"
	"os"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/parsers/lnk"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/parsers/resources"
)

type CustomJumpList struct {
	appId resources.ApplicationId
	path string
	lnks []*lnk.Lnk
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
		path: filePath,
		lnks: lnks,
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

	// The magic number for a LNK file is 0x4c000000.
	lnkSignature := []byte{0x4c, 0x00, 0x00, 0x00}

	// Scan through file looking for LNK signatures
	for i := 0; i < len(fileBytes); i++ {

		// Check if we found a LNK signature
		if i+len(lnkSignature) > len(fileBytes) {
			break
		}

		if !bytes.Equal(fileBytes[i:i+len(lnkSignature)], lnkSignature) {
			continue
		}

		start := i
		// Find end - either next signature or EOF
		end := len(fileBytes)
		for j := start + len(lnkSignature); j < len(fileBytes)-len(lnkSignature); j++ {
			if bytes.Equal(fileBytes[j:j+len(lnkSignature)], lnkSignature) {
				end = j
				break
			}
		}

		// Carve out the LNK file, and convert it to a LnkFile struc usin golnk
		lnkFile, err := lnk.NewLnkFromBytes(fileBytes[start:end], log)
		if err != nil {
			return nil, fmt.Errorf("failed to read LNK file: %w", err)
		}

		// The LnkFile struct is not a complete representation of the jumplist entry
		// We need to do some enrichment, so we wrap it in a JumpListLnk struct.
		lnks = append(lnks, lnkFile)

		i = end - 1 // Move cursor to end (minus 1 since loop will increment)
	}
	return lnks, nil
}



