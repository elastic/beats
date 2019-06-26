// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mage

import (
	"github.com/elastic/beats/dev-tools/mage"
)

// XPackConfigFileParams returns the configuration of sample and reference configuration data.
func XPackConfigFileParams() mage.ConfigFileParams {
	return mage.ConfigFileParams{
		ShortParts: []string{
			mage.OSSBeatDir("_meta/beat.yml"),
			mage.LibbeatDir("_meta/config.yml.tmpl"),
		},
		ReferenceParts: []string{
			mage.OSSBeatDir("_meta/beat.reference.yml"),
			mage.LibbeatDir("_meta/config.reference.yml.tmpl"),
		},
		ExtraVars: map[string]interface{}{
			"ExcludeConsole":    true,
			"ExcludeFileOutput": true,
			"ExcludeKafka":      true,
			"ExcludeLogstash":   true,
			"ExcludeRedis":      true,
		},
	}
}
