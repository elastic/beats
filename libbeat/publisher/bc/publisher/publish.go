package publisher

import (
	"flag"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/publisher/pipeline"
)

// command line flags
var publishDisabled *bool

var debug = logp.MakeDebug("publish")

type ShipperConfig struct {
	// Event processing configurations
	common.EventMetadata `config:",inline"`      // Fields and tags to add to each event.
	Processors           processors.PluginConfig `config:"processors"`

	// Event queue
	Queue common.ConfigNamespace `config:"queue"`
}

func init() {
	publishDisabled = flag.Bool("N", false, "Disable actual publishing for testing")
}

// Create new PublisherType
func New(
	beat common.BeatInfo,
	output common.ConfigNamespace,
	shipper ShipperConfig,
) (*pipeline.Pipeline, error) {
	if *publishDisabled {
		logp.Info("Dry run mode. All output types except the file based one are disabled.")
	}

	processors, err := processors.New(shipper.Processors)
	if err != nil {
		return nil, fmt.Errorf("error initializing processors: %v", err)
	}

	return createPipeline(beat, shipper, processors, output)
}
