// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Shared types are defined here in this package to make structuring better
package gcs

import (
	"time"
)

// Source, it is the cursor source
type Source struct {
	BucketName               string
	BucketTimeOut            time.Duration
	ProjectId                string
	MaxWorkers               int
	Poll                     bool
	PollInterval             time.Duration
	ParseJSON                bool
	TimeStampEpoch           *int64
	FileSelectors            []fileSelectorConfig
	ExpandEventListFromField string
}

func (s *Source) Name() string {
	return s.ProjectId + "::" + s.BucketName
}

const (
	jsonType     = "application/json"
	octetType    = "application/octet-stream"
	ndJsonType   = "application/x-ndjson"
	gzType       = "application/x-gzip"
	encodingGzip = "gzip"
)

var allowedContentTypes = map[string]bool{
	jsonType:   true,
	octetType:  true,
	ndJsonType: true,
	gzType:     true,
}
