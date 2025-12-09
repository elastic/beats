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
)

// ParseCustomJumpListFile parses a custom jump list file into a JumpList object.
// It returns a JumpList object and an error if the file cannot be read or parsed.
func ParseCustomJumpListFile(filePath string, log *logger.Logger) (*JumpList, error) {
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read")
	}

	// Scan through the file looking for footer signatures, there may be multiple custom jump lists in the file
	lnks := carveLnkFiles(fileBytes, log)

	entries := make([]*JumpListEntry, 0)
	for _, lnk := range lnks {
		jumpListEntry := &JumpListEntry{
			Lnk: lnk,
		}
		entries = append(entries, jumpListEntry)
	}

	// If the jumplist file is empty, return an error jlecmd does this as well
	if len(lnks) == 0 {
		return nil, fmt.Errorf("custom jumplist file %s is empty", filePath)
	}

	// Look up the application id and create the metadata
	jumpListMeta := JumpListMeta{
		ApplicationId: GetAppIdFromFileName(filePath, log),
		JumplistType:  JumpListTypeCustom,
		Path:          filePath,
	}
	customJumpList := &JumpList{
		JumpListMeta: jumpListMeta,
		entries:      entries,
	}
	return customJumpList, nil
}

// GetCustomJumpLists finds all the custom jump list files and parses them into JumpList objects.
// It returns a slice of JumpList objects.
func GetCustomJumpLists(log *logger.Logger) []*JumpList {
	files, err := FindJumplistFiles(JumpListTypeCustom, log)
	if err != nil {
		log.Infof("failed to find Custom Jump Lists: %v", err)
		return []*JumpList{}
	}

	var jumplists []*JumpList
	for _, file := range files {
		customJumpList, err := ParseCustomJumpListFile(file, log)
		if err != nil {
			log.Infof("failed to parse Custom Jump List %s: %v", file, err)
			continue
		}
		jumplists = append(jumplists, customJumpList)
	}
	return jumplists
}

// carveLnkFiles scans the fileBytes buffer looking for LNK signatures and carves out the individual LNK files.
// It returns a slice of Lnk objects.
func carveLnkFiles(fileBytes []byte, log *logger.Logger) []*Lnk {
	var lnks []*Lnk
	sigLen := len(LnkSignature)

	// Find the first LNK signature
	start := bytes.Index(fileBytes, LnkSignature)
	if start == -1 {
		return lnks
	}

	// advance the buffer to the first LNK signature
	fileBytes = fileBytes[start:]

	for {
		// Find the next LNK signature
		nextSigIndex := bytes.Index(fileBytes[sigLen:], LnkSignature)

		if nextSigIndex == -1 {
			// This is the last Lnk in the file
			lnk, err := NewLnkFromBytes(fileBytes, log)
			if err == nil {
				lnks = append(lnks, lnk)
			}
			break
		}

		// calculate the cut point for the current Lnk
		// nextSigIndex is a relative index to the start of the fileBytes buffer
		// so we need to add the sigLen to get the absolute index
		cutPoint := nextSigIndex + sigLen
		lnk, err := NewLnkFromBytes(fileBytes[:cutPoint], log)
		if err == nil {
			lnks = append(lnks, lnk)
		}
		// advance the buffer to the next LNK signature
		fileBytes = fileBytes[cutPoint:]
	}
	return lnks
}
