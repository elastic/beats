package module

import (
	"time"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher/bc/publisher"
	"github.com/elastic/beats/metricbeat/mb"
)

// Factory is used to register and reload modules
type Factory struct {
	client        func() publisher.Client
	maxStartDelay time.Duration
}

// NewFactory creates new Reloader instance for the given config
func NewFactory(maxStartDelay time.Duration, p publisher.Publisher) *Factory {
	return &Factory{
		client:        p.Connect,
		maxStartDelay: maxStartDelay,
	}
}

func (r *Factory) Create(c *common.Config) (cfgfile.Runner, error) {
	w, err := NewWrapper(r.maxStartDelay, c, mb.Registry)
	if err != nil {
		return nil, err
	}

	mr := NewRunner(r.client, w)
	return mr, nil
}
