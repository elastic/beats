// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package test

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/metricbeat/beater"
	"github.com/elastic/beats/metricbeat/mb/module"
	xpackbeater "github.com/elastic/beats/x-pack/metricbeat/beater"
)

// BeatCreator creates a customized instance of Metricbeat for the modules test subcommand
func BeatCreator() beat.Creator {
	// Use a customized instance of Metricbeat where startup delay has
	// been disabled to workaround the fact that Modules() will return
	// the static modules (not the dynamic ones) with a start delay.
	return beater.Creator(
		xpackbeater.WithLightModules(),
		beater.WithModuleOptions(
			module.WithMetricSetInfo(),
			module.WithMaxStartDelay(0),
		),
	)
}
