package publisher

import (
	"flag"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/publisher/pipeline"
)

// command line flags
var publishDisabled *bool

var debug = logp.MakeDebug("publish")

type ShipperConfig struct {
	common.EventMetadata `config:",inline"`     // Fields and tags to add to each event.
	Name                 string                 `config:"name"`
	Queue                common.ConfigNamespace `config:"queue"`

	// internal publisher queue sizes
	MaxProcs *int `config:"max_procs"`
}

func init() {
	publishDisabled = flag.Bool("N", false, "Disable actual publishing for testing")
}

// Create new PublisherType
func New(
	beat common.BeatInfo,
	output common.ConfigNamespace,
	shipper ShipperConfig,
	processors *processors.Processors,
) (*pipeline.Pipeline, error) {
	if *publishDisabled {
		logp.Info("Dry run mode. All output types except the file based one are disabled.")
	}

	return createPipeline(beat, shipper, processors, output)
}
