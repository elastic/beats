// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"io"
	"os"
	"strings"

	"github.com/richardlehane/mscfb"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

func ParseAutomaticJumpListFile(filePath string, log *logger.Logger) (*JumpList, error) {

	// Create a minimal JumpList object to return if there is an error.
	automaticJumpList := &JumpList{
		JumpListMeta: JumpListMeta{
			ApplicationId: GetAppIdFromFileName(filePath, log),
			JumplistType:  JumpListTypeAutomatic,
			Path:          filePath,
		},
		entries: []*JumpListEntry{},
	}

	// Open the jumplist file
	file, err := os.Open(filePath)
	if err != nil {
		log.Errorf("failed to open jumplist file %s: %v", filePath, err)
		return automaticJumpList, nil
	}
	defer file.Close()

	// Parse the file as a Microsoft Compound File Binary (OLECFB)
	doc, err := mscfb.New(file)
	if err != nil {
		log.Infof("failed to parse jumplist file %s as OLECFB: %v", filePath, err)
		return automaticJumpList, nil
	}

	// The automatic jumplist is an OLECFB file. It is a collection of entries
	// that contain the jumplist data.  Entries of note are the DestList and DestListPropertyStore streams.
	// The DestList stream contains the list of entries in the jumplist.
	// The DestListPropertyStore stream contains the property store for the jumplist. TODO: Parse this.
	// The other entries are LNK files that contain the destination information for the jumplist entries.
	//
	// This block iterates over the entries in the OLECFB
	//   - when it encounters the DestList and DestListPropertyStore streams, it parses them.
	//   - when it encounters a Lnk file it saves the stream for later parsing
	//   - all other streams are logged as unknown and ignored.
	//
	lnks := make(map[string]*Lnk)
	var destList *DestList
	for entry, err := doc.Next(); err == nil; entry, err = doc.Next() {
		// TODO: Parse the DestListPropertyStore stream.
		if strings.EqualFold(entry.Name, DestListPropertyStoreStreamName) {
			log.Infof("DestListPropertyStore stream found for path %s", filePath)
			continue
		}

		// Parse the DestList stream.
		if strings.EqualFold(entry.Name, DestListStreamName) {
			// Read the DestList stream into a byte slice.
			destListBytes := make([]byte, entry.Size)
			if _, err := io.ReadFull(entry, destListBytes); err != nil {
				log.Infof("failed to read DestList stream for path %s: %v", filePath, err)
				return automaticJumpList, nil
			}

			// Parse the DestList stream into a DestList object.  The DestList is a
			// crucial part of the jumplist, so we can't continue if it fails.
			destList, err = NewDestList(destListBytes, log)
			if err != nil {
				log.Infof("failed to parse DestList for path %s: %v", filePath, err)
				return automaticJumpList, nil
			}
			continue
		}

		// Other than the DestList and DestListPropertyStore streams, we only care about LNK files.
		// Log unknown streams, but continue to the next entry.

		// Read the first 4 bytes of the stream to check if it is a LNK file.
		header := make([]byte, 4)
		if _, err := io.ReadFull(entry, header); err != nil {
			log.Infof("failed to read stream %s for path %s: %v", entry.Name, filePath, err)
			continue
		}

		if !IsLnkSignature(header) {
			log.Infof("stream %s for path %s is not a LNK file", entry.Name, filePath)
			continue
		}

		// Read the rest of the stream into a byte slice.
		streamBuffer := make([]byte, entry.Size)
		copy(streamBuffer, header)
		if _, err := io.ReadFull(entry, streamBuffer[4:]); err != nil {
			log.Infof("failed to read stream %s for path %s: %v", entry.Name, filePath, err)
			continue
		}

		// Parse the LNK stream into a Lnk object.
		lnk, err := NewLnkFromBytes(streamBuffer, log)
		if err != nil {
			log.Infof("failed to parse LNK stream %s for path %s: %v", entry.Name, filePath, err)
			continue
		}

		// Save the Lnk object to the map with a lowercase key for case-insensitive lookup.
		// The lnk object is named in the OLECFB by the hex value of the dest list entry number.
		// We will save it to the map with a lowercase key for case-insensitive lookup.
		lnks[strings.ToLower(entry.Name)] = lnk
	}

	if destList == nil {
		log.Infof("DestList not found for path %s", filePath)
		return automaticJumpList, nil
	}

	// We have a parsed DestList object and a map of Lnk objects.
	// Now we need to associate the Lnk objects with the DestList entries.
	entries := make([]*JumpListEntry, 0, len(destList.Entries))
	for _, entry := range destList.Entries {
		// Create a minimal JumpListEntry object to return if there is an error.
		jumpListEntry := &JumpListEntry{
			DestListEntry: entry,
			Lnk:           nil,
		}

		// Lookup the Lnk object by the DestList entry name.
		lnk, ok := lnks[strings.ToLower(entry.name)]
		if !ok {
			log.Infof("LNK object %s not found for path %s", entry.name, filePath)
			entries = append(entries, jumpListEntry)
			continue
		}
		jumpListEntry.Lnk = lnk
		entries = append(entries, jumpListEntry)
	}

	// Save the entries to the JumpList object.
	automaticJumpList.entries = entries

	return automaticJumpList, nil
}

// GetAutomaticJumpLists finds all the automatic jump list files and parses them into JumpList objects.
// It returns a slice of JumpList objects.
func GetAutomaticJumpLists(log *logger.Logger) []*JumpList {
	files, err := FindJumpListFiles(JumpListTypeAutomatic, log)
	if err != nil {
		log.Infof("failed to find Automatic Jump Lists: %v", err)
		return []*JumpList{}
	}

	var jumplists []*JumpList
	for _, file := range files {
		automaticJumpList, err := ParseAutomaticJumpListFile(file, log)
		if err != nil {
			log.Errorf("failed to parse Automatic Jump List %s: %v", file, err)
			continue
		}
		jumplists = append(jumplists, automaticJumpList)
	}
	return jumplists
}
