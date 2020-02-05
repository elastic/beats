// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mage

import (
	devtools "github.com/elastic/beats/dev-tools/mage"
)

// XPackConfigFileParams returns the configuration of sample and reference configuration data.
func XPackConfigFileParams() devtools.ConfigFileParams {
	return devtools.ConfigFileParams{
		ShortParts: []string{
			devtools.OSSBeatDir("_meta/beat.yml"),
			devtools.LibbeatDir("_meta/config.yml.tmpl"),
		},
		ReferenceParts: []string{
			devtools.OSSBeatDir("_meta/beat.reference.yml"),
			devtools.LibbeatDir("_meta/config.reference.yml.tmpl"),
		},
		ExtraVars: map[string]interface{}{
			"ExcludeConsole":             true,
			"ExcludeFileOutput":          true,
			"ExcludeKafka":               true,
			"ExcludeRedis":               true,
			"UseDockerMetadataProcessor": false,
		},
	}
}
