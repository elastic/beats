// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

/*
Package winlogbeat contains the entrypoint to Winlogbeat which is a lightweight
data shipper for Windows event logs. It ships events directly to Elasticsearch
or Logstash. The data can then be visualized in Kibana.

Downloads: https://www.elastic.co/downloads/beats/winlogbeat
*/
package main

import (
	"os"

	_ "github.com/elastic/beats/winlogbeat/include"

	"github.com/elastic/beats/x-pack/winlogbeat/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
