// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"fmt"
	"os"
	"strings"
	"io"

	"github.com/richardlehane/mscfb"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

// ParseAutomaticJumpListFile parses an automatic jump list file into a JumpList object.
// It returns a JumpList object and an error if the file cannot be read or parsed.
func ParseAutomaticJumpListFile(filePath string, log *logger.Logger) (*JumpList, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Parse the file as a Microsoft Compound File Binary (OLECFB)
	doc, err := mscfb.New(file)
	if err != nil {
		return nil, err
	}

	streams := make(map[string][]byte)
	// Iterate over the entries in the OLECFB
	for entry, err := doc.Next(); err == nil; entry, err = doc.Next() {
		streamName := strings.ToLower(entry.Name)
		streams[streamName], err = io.ReadAll(entry)
		if err != nil {
			return nil, fmt.Errorf("failed to read stream: %w", err)
		}
	}

	// Parse the DestList stream
	destListStream, ok := streams[strings.ToLower(DestListStreamName)]; if !ok {
		return nil, fmt.Errorf("DestList stream not found")
	}
	destList, err := NewDestList(destListStream, log)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DestList: %w", err)
	}

	entries := make([]*JumpListEntry, 0)
	for _, entry := range destList.Entries {
		jumpListEntry := &JumpListEntry{
			DestListEntry: entry,
			Lnk:           nil,
		}

		lnkStream, ok := streams[entry.StreamName]; if !ok {
			fmt.Println("Stream not found: ", entry.StreamName)
			continue
		}

		lnk, err := NewLnkFromBytes(lnkStream, int(entry.EntryNumber), log)
		if err != nil {
			return nil, fmt.Errorf("failed to parse LNK file: %w", err)
		}

		jumpListEntry.Lnk = lnk
		entries = append(entries, jumpListEntry)
	}

	// Look up the application id and create the metadata
	applicationId := NewApplicationIdFromFileName(filePath, log)
	jumpListMeta := JumpListMeta{
		ApplicationId: applicationId,
		JumplistType:  JumpListTypeAutomatic,
		Path:          filePath,
	}
	automaticJumpList := &JumpList{
		JumpListMeta: jumpListMeta,
		entries:      entries,
	}
	return automaticJumpList, nil
}

// GetAutomaticJumpLists finds all the automatic jump list files and parses them into JumpList objects.
// It returns a slice of JumpList objects.
func GetAutomaticJumpLists(log *logger.Logger) []*JumpList {
	files, err := FindJumplistFiles(JumpListTypeAutomatic, log)
	if err != nil {
		log.Infof("failed to find Automatic Jump Lists: %v", err)
		return []*JumpList{}
	}

	var jumplists []*JumpList
	for _, file := range files {
		automaticJumpList, err := ParseAutomaticJumpListFile(file, log)
		if err != nil {
			log.Infof("failed to parse Automatic Jump List %s: %v", file, err)
			continue
		}
		jumplists = append(jumplists, automaticJumpList)
	}
	return jumplists
}
