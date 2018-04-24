package docker

import (
	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/autodiscover/builder"
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/common/docker"
	"github.com/elastic/beats/libbeat/common/safemapstr"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	autodiscover.Registry.AddProvider("docker", AutodiscoverBuilder)
}

// Provider implements autodiscover provider for docker containers
type Provider struct {
	config        *Config
	bus           bus.Bus
	builders      autodiscover.Builders
	appenders     autodiscover.Appenders
	watcher       docker.Watcher
	templates     *template.Mapper
	stop          chan interface{}
	startListener bus.Listener
	stopListener  bus.Listener
}

// AutodiscoverBuilder builds and returns an autodiscover provider
func AutodiscoverBuilder(bus bus.Bus, c *common.Config) (autodiscover.Provider, error) {
	cfgwarn.Beta("The docker autodiscover is beta")
	config := defaultConfig()
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	watcher, err := docker.NewWatcher(config.Host, config.TLS, false)
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

	start := watcher.ListenStart()
	stop := watcher.ListenStop()

	if err := watcher.Start(); err != nil {
		return nil, err
	}

	return &Provider{
		config:        config,
		bus:           bus,
		builders:      builders,
		appenders:     appenders,
		templates:     mapper,
		watcher:       watcher,
		stop:          make(chan interface{}),
		startListener: start,
		stopListener:  stop,
	}, nil
}

// Start the autodiscover process
func (d *Provider) Start() {
	go func() {
		for {
			select {
			case <-d.stop:
				d.startListener.Stop()
				d.stopListener.Stop()
				return

			case event := <-d.startListener.Events():
				d.emitContainer(event, "start")

			case event := <-d.stopListener.Events():
				d.emitContainer(event, "stop")
			}
		}
	}()
}

func (d *Provider) emitContainer(event bus.Event, flag string) {
	container, ok := event["container"].(*docker.Container)
	if !ok {
		logp.Err("Couldn't get a container from watcher event")
		return
	}

	var host string
	if len(container.IPAddresses) > 0 {
		host = container.IPAddresses[0]
	}

	labelMap := common.MapStr{}
	for k, v := range container.Labels {
		safemapstr.Put(labelMap, k, v)
	}

	meta := common.MapStr{
		"container": common.MapStr{
			"id":     container.ID,
			"name":   container.Name,
			"image":  container.Image,
			"labels": labelMap,
		},
	}
	// Without this check there would be overlapping configurations with and without ports.
	if len(container.Ports) == 0 {
		event := bus.Event{
			flag:     true,
			"host":   host,
			"docker": meta,
			"meta": common.MapStr{
				"docker": meta,
			},
		}

		d.publish(event)
	}

	// Emit container container and port information
	for _, port := range container.Ports {
		event := bus.Event{
			flag:     true,
			"host":   host,
			"port":   port.PrivatePort,
			"docker": meta,
			"meta": common.MapStr{
				"docker": meta,
			},
		}

		d.publish(event)
	}
}

func (d *Provider) publish(event bus.Event) {
	// Try to match a config
	if config := d.templates.GetConfig(event); config != nil {
		event["config"] = config
	} else {
		// If no template matches, try builders:
		if config := d.builders.GetConfig(d.generateHints(event)); config != nil {
			event["config"] = config
		}
	}

	// Call all appenders to append any extra configuration
	d.appenders.Append(event)

	d.bus.Publish(event)
}

func (d *Provider) generateHints(event bus.Event) bus.Event {
	// Try to build a config with enabled builders. Send a provider agnostic payload.
	// Builders are Beat specific.
	e := bus.Event{}
	var dockerMeta common.MapStr

	if rawDocker, err := common.MapStr(event).GetValue("docker.container"); err == nil {
		dockerMeta = rawDocker.(common.MapStr)
		e["container"] = dockerMeta
	}

	if host, ok := event["host"]; ok {
		e["host"] = host
	}
	if port, ok := event["port"]; ok {
		e["port"] = port
	}
	if labels, err := dockerMeta.GetValue("labels"); err == nil {
		hints := builder.GenerateHints(labels.(common.MapStr), "", d.config.Prefix)
		e["hints"] = hints
	}
	return e
}

// Stop the autodiscover process
func (d *Provider) Stop() {
	close(d.stop)
}

func (d *Provider) String() string {
	return "docker"
}
