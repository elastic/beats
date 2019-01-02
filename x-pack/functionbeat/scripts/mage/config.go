// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mage

import "github.com/elastic/beats/dev-tools/mage"

// config generates short/reference configs.
func config() error {
	return mage.Config(mage.ShortConfigType|mage.ReferenceConfigType, configFileParams(), ".")
}

func configFileParams() mage.ConfigFileParams {
	return mage.ConfigFileParams{
		ShortParts: []string{
			mage.OSSBeatDir("_meta/beat.yml"),
			mage.LibbeatDir("_meta/config.yml"),
		},
		ReferenceParts: []string{
			mage.OSSBeatDir("_meta/beat.reference.yml"),
			mage.LibbeatDir("_meta/config.reference.yml"),
		},
	}
}
