// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package parsers

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"github.com/parsiya/golnk"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

type JumpListLnk struct {
	lnk.LnkFile
	AppId ApplicationId
	ShellItems []ShellItem
}

func NewJumpListLnk(lnkFile lnk.LnkFile, appId ApplicationId) JumpListLnk {
	jumpListLnk := JumpListLnk{LnkFile: lnkFile, AppId: appId}
	for _, item := range lnkFile.IDList.List.ItemIDList {
		jumpListLnk.ShellItems = append(jumpListLnk.ShellItems, ShellItem{Data: item.Data})
	}
	return jumpListLnk
}

var isHexString = regexp.MustCompile(`^[0-9a-fA-F]+$`).MatchString

func GetAppIdFromFileName(filePath string, log *logger.Logger) ApplicationId {
	fileName := filepath.Base(filePath)
	dotIndex := strings.Index(fileName, ".")
	if dotIndex != -1 {
		return NewApplicationId(fileName[:dotIndex])
	}
	return NewApplicationId("")
}

func ParseCustomDestination(filePath string, log *logger.Logger) ([]JumpListLnk, error) {
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var lnkFiles []JumpListLnk
	lnkSignature := []byte{0x4c, 0x00, 0x00, 0x00}

	appId := GetAppIdFromFileName(filePath, log)

	// Scan through file looking for LNK signatures
	for i := 0; i < len(fileBytes); i++ {
		// Check if we found a LNK signature
		if i+len(lnkSignature) <= len(fileBytes) &&
			bytes.Equal(fileBytes[i:i+len(lnkSignature)], lnkSignature) {

			start := i
			// Find end - either next signature or EOF
			end := len(fileBytes)
			for j := start + len(lnkSignature); j < len(fileBytes)-len(lnkSignature); j++ {
				if bytes.Equal(fileBytes[j:j+len(lnkSignature)], lnkSignature) {
					end = j
					break
				}
			}

			// Extract the LNK file bytes
			lnkFile, err := lnk.Read(bytes.NewReader(fileBytes[start:end]), uint64(end-start))
			if err != nil {
				return nil, err
			}
			lnkFiles = append(lnkFiles, NewJumpListLnk(lnkFile, appId))
			i = end - 1 // Move cursor to end (minus 1 since loop will increment)
		}
	}
	if len(lnkFiles) == 0 {
		log.Errorf("custom destination %s: empty link file", filePath)
		return nil, fmt.Errorf("custom destination %s: empty link file", filePath)
	}
	return lnkFiles, nil
}
