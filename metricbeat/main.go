// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
	"github.com/elastic/beats/v7/metricbeat/cmd"
	"github.com/elastic/beats/v7/metricbeat/module/systemtest"
)

func main() {
	rootCmd := cmd.RootCmd(beater.WithV2Inputs(v2Inputs))
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func v2Inputs(info beat.Info, log *logp.Logger) []v2.Plugin {
	flatten := func(lists ...[]v2.Plugin) []v2.Plugin {
		var inputs []v2.Plugin
		for _, l := range lists {
			inputs = append(inputs, l...)
		}
		return inputs
	}

	return flatten(
		systemtest.Inputs(),
	)
}
