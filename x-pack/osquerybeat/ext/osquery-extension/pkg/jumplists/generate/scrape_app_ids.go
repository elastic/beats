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

const appIdSourceUrl = "https://raw.githubusercontent.com/EricZimmerman/JumpList/refs/heads/master/JumpList/Resources/AppIDs.txt"
const appIdCachePath = "generate/sources/AppIDs.txt"

// scrapeJumplistAppIDs pulls the app ids from appSourceUrl and returns a map of app ids to app names.
// the app ids are in the format of a hex string, and the app names are in the format of a string.
func scrapeJumplistAppIDs(opts generatorOptions, log *logger.Logger) (map[string]string, error) {
	appIDs := make(map[string]string)
	valueRegex := regexp.MustCompile(`.*"(.*)"`)
	bodyString, err := loadSourceText(
		filepath.Join(opts.workingDir, appIdCachePath),
		appIdSourceUrl,
		opts.refreshSources,
		log,
	)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(bodyString, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue // Skip empty lines
		}

		// app_id_raw, app_name_raw = line.strip().split('|')
		parts := strings.Split(line, "|")
		if len(parts) != 2 {
			log.Infof("invalid line in response: %s", line)
			continue
		}
		appIDRaw := parts[0]
		appNameRaw := parts[1]

		appIDMatches := valueRegex.FindStringSubmatch(appIDRaw)
		appNameMatches := valueRegex.FindStringSubmatch(appNameRaw)

		if len(appIDMatches) < 2 || len(appNameMatches) < 2 {
			log.Infof("failed to parse quoted values from response: %s", line)
			continue
		}

		appID := appIDMatches[1]
		appName := appNameMatches[1]

		if isHexString.MatchString(appID) {
			appIDs[strings.ToLower(appID)] = appName
		} else {
			log.Infof("Invalid app id in response: %s", line)
		}
	}

	return appIDs, nil
}

// writeAppIdGeneratedFile writes the app ids to a generated source file
// that can be used to lookup app ids by name.
func writeAppIdGeneratedFile(outputFile string, opts generatorOptions, log *logger.Logger) error {
	appIDs, err := scrapeJumplistAppIDs(opts, log)
	if err != nil {
		return err
	}

	// Build the Go map literal string
	var sb strings.Builder
	writeCopyrightHeader(&sb)
	sb.WriteString("// knownAppIDs is a lookup table for known windows AppIDs.\n")
	sb.WriteString("// Source Repo:         https://github.com/EricZimmerman/JumpList/\n")
	sb.WriteString("// Source Repo License: MIT License (https://github.com/EricZimmerman/JumpList/blob/master/license)\n")
	sb.WriteString("var knownAppIDs = map[string]string{\n")

	// Sort keys for consistent output
	keys := make([]string, 0, len(appIDs))
	for k := range appIDs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, appID := range keys {
		appName := appIDs[appID]
		sb.WriteString(fmt.Sprintf("\t%-20s: %q,\n", fmt.Sprintf("\"%s\"", appID), appName))
	}
	sb.WriteString("}\n")

	err = os.WriteFile(outputFile, []byte(sb.String()), 0o644)
	if err != nil {
		return err
	}

	log.Infof("Generated %s successfully", outputFile)
	return nil
}
