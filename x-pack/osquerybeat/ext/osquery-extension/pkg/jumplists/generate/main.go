// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package main

import (
	"os"
	"path/filepath"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

func main() {
	log := logger.New(os.Stdout, true)

	// Files will be written to the directory containing the go generate directive
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get working directory: %v", err)
	}

	// Write the generated app IDs file
	appIdsFile := filepath.Join(wd, "generated_app_ids.go")
	err = writeAppIdGeneratedFile(appIdsFile, log)
	if err != nil {
		log.Fatalf("failed to write generated_app_ids.go: %v", err)
	}

	// Write the generated guid mappings file
	guidMappingsFile := filepath.Join(wd, "generated_guid_mappings.go")
	err = writeGuidMappingGeneratedFile(guidMappingsFile, log)
	if err != nil {
		log.Fatalf("failed to write generated_guid_mappings.go: %v", err)
	}
}
