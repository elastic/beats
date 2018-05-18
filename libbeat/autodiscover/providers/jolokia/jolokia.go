package jolokia

import (
	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
)

func init() {
	autodiscover.Registry.AddProvider("jolokia", AutodiscoverBuilder)
}

type Provider struct {
	config    *Config
	bus       bus.Bus
	builders  autodiscover.Builders
	appenders autodiscover.Appenders
	templates *template.Mapper
	discovery *Discovery
}

func AutodiscoverBuilder(bus bus.Bus, c *common.Config) (autodiscover.Provider, error) {
	cfgwarn.Experimental("The Jolokia Discovery autodiscover is experimental")

	config := defaultConfig()
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	// Avoid having sockets open more time than needed
	if config.ProbeTimeout > config.Period {
		config.ProbeTimeout = config.Period
	}

	discovery := &Discovery{
		Interfaces:   config.Interfaces,
		Period:       config.Period,
		GracePeriod:  config.GracePeriod,
		ProbeTimeout: config.ProbeTimeout,
	}

	mapper, err := template.NewConfigMapper(config.Templates)
	if err != nil {
		return nil, err
	}

	builders, err := autodiscover.NewBuilders(config.Builders, false)
	if err != nil {
		return nil, err
	}

	appenders, err := autodiscover.NewAppenders(config.Appenders)
	if err != nil {
		return nil, err
	}

	return &Provider{
		bus:       bus,
		templates: mapper,
		builders:  builders,
		appenders: appenders,
		discovery: discovery,
	}, nil
}

func (p *Provider) Start() {
	p.discovery.Start()
	go func() {
		for event := range p.discovery.Events() {
			p.publish(event.BusEvent())
		}
	}()
}

func (p *Provider) publish(event bus.Event) {
	if config := p.templates.GetConfig(event); config != nil {
		event["config"] = config
	} else if config := p.builders.GetConfig(event); config != nil {
		event["config"] = config
	}

	p.appenders.Append(event)

	p.bus.Publish(event)
}

func (p *Provider) Stop() {
	p.discovery.Stop()
}

func (p *Provider) String() string {
	return "jolokia"
}
