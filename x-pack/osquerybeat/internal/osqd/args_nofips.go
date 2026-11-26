// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package osqd

import "github.com/elastic/beats/v7/pkg/logp"

// enforceFIPSFlags returns flags unchanged in builds without the requirefips
// tag.
func enforceFIPSFlags(flags Flags, _ *logp.Logger) Flags {
	return flags
}

// EnforceFIPSOptions returns options unchanged in builds without the
// requirefips tag.
func EnforceFIPSOptions(options map[string]any) map[string]any {
	return options
}
