/*
Package metricbeat contains the entrypoint to Metricbeat which is a lightweight
data shipper for operating system and service metrics. It ships events directly
to Elasticsearch or Logstash. The data can then be visualized in Kibana.

Downloads: https://www.elastic.co/downloads/beats/metricbeat
*/
package main

import (
	"os"

	"github.com/elastic/beats/metricbeat/beater"
	_ "github.com/elastic/beats/metricbeat/include"

	"github.com/elastic/beats/libbeat/beat"
)

// Name of this Beat.
var Name = "metricbeat"

func main() {
	if err := beat.Run(Name, "", beater.New); err != nil {
		os.Exit(1)
	}
}
