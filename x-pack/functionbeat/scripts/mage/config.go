// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mage

import (
	devtools "github.com/menderesk/beats/v7/dev-tools/mage"
)

// XPackConfigFileParams returns the configuration of sample and reference configuration data.
func XPackConfigFileParams() devtools.ConfigFileParams {
	p := devtools.DefaultConfigFileParams()
	p.Templates = append(p.Templates, "_meta/config/*.tmpl")
	p.ExtraVars = map[string]interface{}{
		"ExcludeConsole":             false,
		"ExcludeFileOutput":          true,
		"ExcludeKafka":               true,
		"ExcludeRedis":               true,
		"UseDockerMetadataProcessor": false,
	}
	return p
}
