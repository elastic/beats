/*
Package winlogbeat contains the entrypoint to Winlogbeat which is a lightweight
data shipper for Windows event logs. It ships events directly to Elasticsearch
or Logstash. The data can then be visualized in Kibana.

Downloads: https://www.elastic.co/downloads/beats/winlogbeat
*/
package main

import (
	"os"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/winlogbeat/beater"
)

// Name of this beat.
var Name = "winlogbeat"

func main() {
	if err := beat.Run(Name, "", beater.New); err != nil {
		os.Exit(1)
	}
}
