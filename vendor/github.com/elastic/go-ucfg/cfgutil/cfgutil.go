package cfgutil

import "github.com/elastic/go-ucfg"

// Collector collects and merges multiple generated *ucfg.Config, remembering
// errors, for postponing error checking after having merged all loaded configurations.
type Collector struct {
	config *ucfg.Config
	err    error
	opts   []ucfg.Option
}

func NewCollector(cfg *ucfg.Config, opts ...ucfg.Option) *Collector {
	if cfg == nil {
		cfg = ucfg.New()
	}
	return &Collector{config: cfg, err: nil}
}

func (c *Collector) GetOptions() []ucfg.Option {
	return c.opts
}

func (c *Collector) Get() (*ucfg.Config, error) {
	return c.config, c.err
}

func (c *Collector) Config() *ucfg.Config {
	return c.config
}

func (c *Collector) Error() error {
	return c.err
}

func (c *Collector) Add(cfg *ucfg.Config, err error) error {
	if c.err != nil {
		return c.err
	}

	if err != nil {
		c.err = err
		return err
	}

	if cfg != nil {
		err = c.config.Merge(cfg, c.opts...)
		if err != nil {
			c.err = err
		}
	}
	return err
}
