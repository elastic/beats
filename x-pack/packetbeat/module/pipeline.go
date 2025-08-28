// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package module

import (
	"embed"

	"github.com/elastic/beats/v7/packetbeat/module"
)

// pipelineFS holds the yml representation of the ingest node pipelines
//
//go:embed */ingest/*.yml
var pipelinesFS embed.FS

func init() {
	module.PipelinesFS = &pipelinesFS
}
