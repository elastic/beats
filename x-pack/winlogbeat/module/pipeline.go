// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package module

import (
	"embed"

	"github.com/elastic/beats/v8/winlogbeat/module"
)

// pipelineFS holds the yml representation of the ingest node pipelines
//go:embed */ingest/*.yml
var pipelinesFS embed.FS

func Init() {
	module.PipelinesFS = &pipelinesFS
}
