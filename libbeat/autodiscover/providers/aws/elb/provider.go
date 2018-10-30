package elb

import (
	"time"

	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
)

func init() {
	autodiscover.Registry.AddProvider("aws_elb", AutodiscoverBuilder)
}

// Provider implements autodiscover provider for aws ELBs
type Provider struct {
	config        *Config
	bus           bus.Bus
	builders      autodiscover.Builders
	appenders     autodiscover.Appenders
	templates     *template.Mapper
	stop          chan interface{}
	startListener bus.Listener
	stopListener  bus.Listener
	onStop        func()
}

// AutodiscoverBuilder builds and returns an autodiscover provider
func AutodiscoverBuilder(bus bus.Bus, c *common.Config) (autodiscover.Provider, error) {
	cfgwarn.Beta("aws_elb autodiscover is beta")
	config := defaultConfig()
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	mapper, err := template.NewConfigMapper(config.Templates)
	if err != nil {
		return nil, err
	}

	builders, err := autodiscover.NewBuilders(config.Builders, config.HintsEnabled)
	if err != nil {
		return nil, err
	}

	appenders, err := autodiscover.NewAppenders(config.Appenders)
	if err != nil {
		return nil, err
	}

	return &Provider{
		config:    config,
		bus:       bus,
		builders:  builders,
		appenders: appenders,
		templates: mapper,
		stop:      make(chan interface{}),
	}, nil
}

// Start the autodiscover process
func (p *Provider) Start() {
	p.onStop = watch(
		p.config.Region,
		10*time.Second,
		func(uuid string, lb common.MapStr) {
			e := bus.Event{
				"start":   true,
				"hashKey": uuid,
				"host":    lb["host"],
				"port":    lb["port"],
				"meta": common.MapStr{
					"elb": lb,
				},
			}
			if configs := p.templates.GetConfig(e); configs != nil {
				e["config"] = configs
			}
			p.appenders.Append(e)
			p.bus.Publish(e)
		},
		func(arn string) {
			e := bus.Event{
				"stop":    true,
				"hashKey": arn,
			}
			p.bus.Publish(e)
		},
	)
}

// Stop the autodiscover process
func (p *Provider) Stop() {
	close(p.stop)
}

func (p *Provider) String() string {
	return "aws_elb"
}
