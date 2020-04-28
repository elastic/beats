// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package common

import (
	"os"

	"github.com/elastic/beats/v7/libbeat/common"
)

type Parser interface {
	Identify(header []byte) bool
	Parse(f *os.File) (common.MapStr, error)
}

type ParserFactory func() Parser
