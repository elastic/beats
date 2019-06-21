// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"github.com/elastic/beats/libbeat/paths"
	"github.com/elastic/beats/metricbeat/beater"
	"github.com/elastic/beats/metricbeat/mb"
	xpackmb "github.com/elastic/beats/x-pack/metricbeat/mb"
)

// WithLightModules enables light modules support
func WithLightModules() beater.Option {
	return func(*beater.Metricbeat) {
		path := paths.Resolve(paths.Home, "module")
		mb.Registry.SetSecondarySource(xpackmb.NewLightModulesSource(path))
	}
}
