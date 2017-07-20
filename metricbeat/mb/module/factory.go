package module

import (
	"time"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher/bc/publisher"
	"github.com/elastic/beats/metricbeat/mb"
)

// Factory creates new Runner instances from configuration objects.
// It is used to register and reload modules.
type Factory struct {
	pipeline      publisher.Publisher
	maxStartDelay time.Duration
}

// NewFactory creates new Reloader instance for the given config
func NewFactory(maxStartDelay time.Duration, p publisher.Publisher) *Factory {
	return &Factory{
		pipeline:      p,
		maxStartDelay: maxStartDelay,
	}
}

func (r *Factory) Create(c *common.Config) (cfgfile.Runner, error) {
	connector, err := NewConnector(r.pipeline, c)
	if err != nil {
		return nil, err
	}

	w, err := NewWrapper(r.maxStartDelay, c, mb.Registry)
	if err != nil {
		return nil, err
	}

	client, err := connector.Connect()
	if err != nil {
		return nil, err
	}

	mr := NewRunner(client, w)
	return mr, nil
}
