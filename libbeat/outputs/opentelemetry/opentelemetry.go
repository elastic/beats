package opentelemetry

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/outputs"
)

func init() {
	outputs.RegisterType("opentelementry", makeOtel)
}

// makeFileout instantiates a new file output instance.
func makeOtel(
	_ outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return outputs.Fail(err)
	}
	if !cfg.HasField("index") {
		cfg.SetString("index", -1, beat.Beat)
	}

	client := newClient(observer, config.Timeout)
	return outputs.Success(-1, 0, client)
}
