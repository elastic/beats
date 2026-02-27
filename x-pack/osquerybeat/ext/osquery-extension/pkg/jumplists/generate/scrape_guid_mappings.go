// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

const guidMappingSourceUrl = "https://github.com/EricZimmerman/GuidMapping/raw/refs/heads/master/Resources/GuidToName.txt"
const guidMappingCachePath = "generate/sources/GuidToName.txt"

// scrapeGuidMappings
func scrapeGuidMappings(opts generatorOptions, log *logger.Logger) (map[string]string, error) {
	guidMappings := make(map[string]string)
	bodyString, err := loadSourceText(
		filepath.Join(opts.workingDir, guidMappingCachePath),
		guidMappingSourceUrl,
		opts.refreshSources,
		log,
	)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`(?m)^([a-fA-F0-9-]{36})\|(.+)$`)
	matches := re.FindAllStringSubmatch(bodyString, -1)
	for _, match := range matches {
		if len(match) != 3 {
			continue
		}
		guid := strings.ToUpper(match[1])
		guidName := match[2]
		guidMappings[guid] = strings.TrimRight(guidName, "\r\n")
	}

	return guidMappings, nil
}

// writeGuidMappingGeneratedFile writes the guid mappings to a generated source file
// that can be used to lookup guid names by guid.
func writeGuidMappingGeneratedFile(outputFile string, opts generatorOptions, log *logger.Logger) error {
	guidMappings, err := scrapeGuidMappings(opts, log)
	if err != nil {
		return err
	}

	// Build the Go map literal string
	// Using strings.Builder is more efficient than repeated string concatenation
	var sb strings.Builder
	writeCopyrightHeader(&sb)
	sb.WriteString("// knownFolderMappings is a lookup table  for known windows GUIDs to names.\n")
	sb.WriteString("// Source Repo:         https://github.com/EricZimmerman/GuidMapping/\n")
	sb.WriteString("// Source Repo License: MIT License (https://github.com/EricZimmerman/GuidMapping/blob/master/LICENSE)\n")
	sb.WriteString("var knownFolderMappings = map[string]string{\n")

	// Sort keys for consistent output
	keys := make([]string, 0, len(guidMappings))
	for k := range guidMappings {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, guid := range keys {
		guidName := guidMappings[guid]
		fmt.Fprintf(&sb, "    %q : %q,\n", guid, guidName)
	}
	sb.WriteString("}\n")

	err = os.WriteFile(outputFile, []byte(sb.String()), 0o644)
	if err != nil {
		return err
	}

	log.Infof("Generated %s successfully", outputFile)
	return nil
}
