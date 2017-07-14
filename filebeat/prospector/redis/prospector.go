package redis

import (
	"time"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/filebeat/harvester"
	rd "github.com/garyburd/redigo/redis"
)

// Prospector is a prospector for redis
type Prospector struct {
	started  bool
	outlet   channel.Outleter
	config   config
	cfg      *common.Config
	registry *harvester.Registry
}

// NewProspector creates a new redis prospector
func NewProspector(cfg *common.Config, outletFactory channel.OutleterFactory) (*Prospector, error) {

	logp.Experimental("Redis slowlog prospector is enabled.")
	config := defaultConfig

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}

	outlet, err := outletFactory(cfg)
	if err != nil {
		return nil, err
	}

	p := &Prospector{
		started:  false,
		outlet:   outlet,
		config:   config,
		cfg:      cfg,
		registry: harvester.NewRegistry(),
	}

	return p, nil
}

// LoadStates loads the states
func (p *Prospector) LoadStates(states []file.State) error {
	return nil
}

// Run runs the prospector
func (p *Prospector) Run() {
	logp.Debug("redis", "Run redis prospector with hosts: %+v", p.config.Hosts)

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

		p.registry.Start(h)
	}
}

// Stop stopps the prospector and all its harvesters
func (p *Prospector) Stop() {
	p.registry.Stop()
	p.outlet.Close()
}

// Wait waits for the propsector to be completed. Not implemented.
func (p *Prospector) Wait() {}

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
