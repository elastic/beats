// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

/*
Package metricbeat contains the entrypoint to Metricbeat which is a lightweight
data shipper for operating system and service metrics. It ships events directly
to Elasticsearch or Logstash. The data can then be visualized in Kibana.

Downloads: https://www.elastic.co/downloads/beats/metricbeat
*/
package main

import (
	"os"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/beater"
	"github.com/elastic/beats/v7/metricbeat/module/systemtest"
	inputs "github.com/elastic/beats/v7/x-pack/filebeat/input/default-inputs"
	"github.com/elastic/beats/v7/x-pack/metricbeat/cmd"
)

func main() {
	rootCmd := cmd.RootCmd(beater.WithV2Inputs(v2Inputs))
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func v2Inputs(info beat.Info, log *logp.Logger, store beat.StateStore) []v2.Plugin {
	return v2.ConcatPlugins(
		systemtest.Inputs(),           // custom metricset based set of inputs
		inputs.Init(info, log, store), // include x-pack/filebeat inputs
	)
}
