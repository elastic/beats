// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package olecfb

import (
	"fmt"
	"io"
	"os"

	"github.com/richardlehane/mscfb"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/parsers/lnk"
)

// DestListStreamName is the name of the stream that contains the destination list.
const DestListStreamName = "DestList"
// DestListPropertyStoreStreamName is the name of the stream that contains the destination list property store.
const DestListPropertyStoreStreamName = "DestListPropertyStore"


// Olecfb represents a microsoft compound file binary file.
// tailored for jump list files.
// - path: the path to the file
// - streams: a map of stream names to their data
type Olecfb struct {
	Path    string
	DestList *DestList
	Lnks []*lnk.Lnk
	UnknownStreams map[string][]byte
}

func (o *Olecfb) HasValidDestList() bool {
	return o.DestList != nil
}


// NewOlecfb creates a new Olecfb object.
// - path: the path to the file
// - log: the logger to use
// returns: a new Olecfb object, or an error if the file cannot be opened or parsed
func NewOlecfb(path string, log *logger.Logger) (*Olecfb, error) {
	olecfb := &Olecfb{
		Path:    path,
		DestList: nil,
		Lnks: make([]*lnk.Lnk, 0),
		UnknownStreams: make(map[string][]byte),
	}

	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Parse the file as a Microsoft Compound File Binary (OLECFB)
	doc, err := mscfb.New(file)
	if err != nil {
		return nil, err
	}

	// Iterate over the entries in the OLECFB
	for entry, err := doc.Next(); err == nil; entry, err = doc.Next() {
		data, err := io.ReadAll(entry)
		if err != nil {
			return nil, fmt.Errorf("failed to read DestList stream: %w", err)
		}

		// If the entry is a DestList stream, parse it.
		if entry.Name == DestListStreamName {
			// having an empty DestList is not an error, so it will just
			// be nil if the DestList is not valid.
			destList, err := NewDestList(data, log)
			if err == nil {
				olecfb.DestList = destList
			}
			continue
		}

		// If the entry is a LNK file, parse it.
		if lnk.IsLnkSignature(data) {
			lnk, err := lnk.NewLnkFromBytes(data, log)
			if err != nil {
				return nil, fmt.Errorf("failed to parse LNK file: %w", err)
			}
			olecfb.Lnks = append(olecfb.Lnks, lnk)
			continue
		}
		// If the entry is not a DestList stream or a LNK file, store the data in the UnknownStreams map.
		olecfb.UnknownStreams[entry.Name] = data
	}
	return olecfb, nil
}
