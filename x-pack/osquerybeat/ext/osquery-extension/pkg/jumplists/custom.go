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
	jumpliststypes "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/jumplists"
)

// parseCustomJumplistFile parses a custom jump list file into a jumplist object.
// It returns a jumplist object and an error if the file cannot be read or parsed.
// Custom jumplists are comprised of some metadata and a collection of Lnk objects.
// The lnk objects have to be carved out of the file and there may be multiple of them per file
func parseCustomJumplistFile(filePath string, userProfile *UserProfile, log *logger.Logger) (*jumplist, error) {
	// Read the file into a byte slice
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filePath, err)
	}

	// Carve out the Lnk objects from the file
	lnks := carveLnkFiles(fileBytes, log)

	// If the jumplist file is empty, return an error jlecmd does this as well
	if len(lnks) == 0 {
		return nil, fmt.Errorf("custom jumplist file %s is empty", filePath)
	}

	// Look up the application id and create the metadata
	jumpListMeta := &jumplistMeta{
		UserProfile:   &jumpliststypes.UserProfile{Username: userProfile.Username, Sid: userProfile.Sid},
		ApplicationID: getAppIdFromFileName(filePath),
		JumplistMeta:  &jumpliststypes.JumplistMeta{JumplistType: string(jumplistTypeCustom), SourceFilePath: filePath},
	}
	entries := make([]*jumplistEntry, 0, len(lnks))
	for _, lnk := range lnks {
		entries = append(entries, &jumplistEntry{Lnk: lnk})
	}

	// Combine the metadata and the entries into a Jumplist object
	customJumplist := &jumplist{
		jumplistMeta: jumpListMeta,
		entries:      entries,
	}
	return customJumplist, nil
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
			lnk, err := newLnkFromBytes(fileBytes, log)
			if err == nil {
				lnks = append(lnks, lnk)
			}
			break
		}

		// calculate the cut point for the current Lnk
		// nextSigIndex is a relative index to the start of the fileBytes buffer
		// so we need to add the sigLen to get the absolute index
		cutPoint := nextSigIndex + sigLen
		lnk, err := newLnkFromBytes(fileBytes[:cutPoint], log)
		if err == nil {
			lnks = append(lnks, lnk)
		}
		// advance the buffer to the next LNK signature
		fileBytes = fileBytes[cutPoint:]
	}
	return lnks
}
