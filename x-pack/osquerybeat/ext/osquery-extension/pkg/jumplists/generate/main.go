// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

type generatorOptions struct {
	workingDir      string
	refreshSources  bool
}

func main() {
	log := logger.New(os.Stdout, true)
	refreshSources := flag.Bool("refresh-sources", false, "Download latest source files and update local cached copies")
	flag.Parse()

	// Files will be written to the directory containing the go generate directive
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get working directory: %v", err)
	}

	opts := generatorOptions{
		workingDir:     wd,
		refreshSources: *refreshSources,
	}

	// Write the generated app IDs file
	appIdsFile := filepath.Join(wd, "generated_app_ids.go")
	err = writeAppIdGeneratedFile(appIdsFile, opts, log)
	if err != nil {
		log.Fatalf("failed to write generated_app_ids.go: %v", err)
	}

	// Write the generated guid mappings file
	guidMappingsFile := filepath.Join(wd, "generated_guid_mappings.go")
	err = writeGuidMappingGeneratedFile(guidMappingsFile, opts, log)
	if err != nil {
		log.Fatalf("failed to write generated_guid_mappings.go: %v", err)
	}
}
