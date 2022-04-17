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

	"github.com/menderesk/beats/v7/x-pack/metricbeat/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
