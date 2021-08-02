// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/metricbeat/beater"
	mbcmd "github.com/elastic/beats/v7/metricbeat/cmd"

	// Register the includes.
	_ "github.com/elastic/beats/v7/x-pack/libbeat/include"
	_ "github.com/elastic/beats/v7/x-pack/metricbeat/include"

	// Import OSS modules.
	_ "github.com/elastic/beats/v7/metricbeat/include"
	_ "github.com/elastic/beats/v7/metricbeat/include/fields"
)

// RootCmd to handle beats cli
func RootCmd(opts ...beater.Option) *cmd.BeatsRootCmd {
	settings := mbcmd.MetricbeatSettings()
	settings.ElasticLicensed = true
	return mbcmd.Initialize(settings, opts...)
}
