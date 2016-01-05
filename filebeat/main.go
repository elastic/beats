package main

import (
	filebeat "github.com/elastic/beats/filebeat/beat"
	"github.com/elastic/beats/libbeat/beat"
)

var Name = "filebeat"

// The basic model of execution:
// - prospector: finds files in paths/globs to harvest, starts harvesters
// - harvester: reads a file, sends events to the spooler
// - spooler: buffers events until ready to flush to the publisher
// - publisher: writes to the network, notifies registrar
// - registrar: records positions of files read
// Finally, prospector uses the registrar information, on restart, to
// determine where in each file to restart a harvester.

func main() {
	beat.Run(Name, "", filebeat.New())
}
