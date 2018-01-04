package main

import (
	"github.com/elastic/beats/libbeat/plugin"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/processors/add_process_metadata"
)

var Bundle = plugin.Bundle(
	processors.Plugin("add_process_metadata", add_process_metadata.New),
)
