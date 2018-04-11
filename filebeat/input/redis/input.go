package redis

import (
	"time"

	rd "github.com/garyburd/redigo/redis"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	err := input.Register("redis", NewInput)
	if err != nil {
		panic(err)
	}
}

// Input is a input for redis
type Input struct {
	started  bool
	outlet   channel.Outleter
	config   config
	cfg      *common.Config
	registry *harvester.Registry
}

// NewInput creates a new redis input
func NewInput(cfg *common.Config, outletFactory channel.Factory, context input.Context) (input.Input, error) {
	cfgwarn.Experimental("Redis slowlog input is enabled.")

	config := defaultConfig

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}

	outlet, err := outletFactory(cfg, context.DynamicFields)
	if err != nil {
		return nil, err
	}

	p := &Input{
		started:  false,
		outlet:   outlet,
		config:   config,
		cfg:      cfg,
		registry: harvester.NewRegistry(),
	}

	return p, nil
}

// LoadStates loads the states
func (p *Input) LoadStates(states []file.State) error {
	return nil
}

// Run runs the input
func (p *Input) Run() {
	logp.Debug("redis", "Run redis input with hosts: %+v", p.config.Hosts)

	if len(p.config.Hosts) == 0 {
		logp.Err("No redis hosts configured")
		return
	}

	forwarder := harvester.NewForwarder(p.outlet)
	for _, host := range p.config.Hosts {
		pool := CreatePool(host, p.config.Password, p.config.Network,
			p.config.MaxConn, p.config.IdleTimeout, p.config.IdleTimeout)

		h := NewHarvester(pool.Get())
		h.forwarder = forwarder

		if err := p.registry.Start(h); err != nil {
			logp.Err("Harvester start failed: %s", err)
		}
	}
}

// Stop stops the input and all its harvesters
func (p *Input) Stop() {
	p.registry.Stop()
	p.outlet.Close()
}

// Wait waits for the input to be completed. Not implemented.
func (p *Input) Wait() {}

// CreatePool creates a redis connection pool
// NOTE: This code is copied from the redis pool handling in metricbeat
func CreatePool(
	host, password, network string,
	maxConn int,
	idleTimeout, connTimeout time.Duration,
) *rd.Pool {
	return &rd.Pool{
		MaxIdle:     maxConn,
		IdleTimeout: idleTimeout,
		Dial: func() (rd.Conn, error) {
			c, err := rd.Dial(network, host,
				rd.DialConnectTimeout(connTimeout),
				rd.DialReadTimeout(connTimeout),
				rd.DialWriteTimeout(connTimeout))
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
	}
}
