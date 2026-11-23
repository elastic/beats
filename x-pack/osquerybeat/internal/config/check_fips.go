// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build requirefips

package config

import (
	"errors"
	"strings"
)

// CheckFIPS returns an error if the inputs use settings that FIPS builds do
// not support. FIPS builds reject these settings instead of ignoring them, so
// that nobody assumes a custom binary or extension is covered by the build.
//
// The settings make osqueryd run code that the FIPS build does not control:
// elastic_options.install replaces the bundled osqueryd, which the enforced
// flags, disabled tables and config options apply to, with a downloaded one.
// elastic_options.extensions.paths and options.extension make osqueryd start
// other programs as extensions, and those add code that the enforced settings
// do not govern. A rejection can be relaxed later without breaking anyone,
// which tightening a restriction cannot.
//
// Every input is checked, not only the result input that the runtime reads.
// The error is a single line, because it ends up in logs and in the status
// that the agent reports for the policy.
func CheckFIPS(inputs []InputConfig) error {
	var extensionPaths, install, extensionFlag bool
	for _, in := range inputs {
		if in.Osquery == nil {
			continue
		}
		if _, ok := in.Osquery.Options["extension"]; ok {
			extensionFlag = true
		}
		options := in.Osquery.ElasticOptions
		if options == nil {
			continue
		}
		if options.Extensions != nil && len(options.Extensions.Paths) != 0 {
			extensionPaths = true
		}
		if options.Install != nil && options.Install.Enabled() {
			install = true
		}
	}

	var unsupported []string
	if extensionPaths {
		unsupported = append(unsupported, "osquery.elastic_options.extensions.paths is not supported in FIPS builds: extensions are code that the build does not cover")
	}
	if install {
		unsupported = append(unsupported, "osquery.elastic_options.install is not supported in FIPS builds: it replaces the bundled osqueryd")
	}
	if extensionFlag {
		unsupported = append(unsupported, "osquery.options.extension is not supported in FIPS builds: it starts a program as an extension")
	}
	if len(unsupported) == 0 {
		return nil
	}
	return errors.New(strings.Join(unsupported, "; "))
}
