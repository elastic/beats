package docker

import (
	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/docker"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	autodiscover.ProviderRegistry.AddProvider("docker", AutodiscoverBuilder)
}

// Provider implements autodiscover provider for docker containers
type Provider struct {
	config        *Config
	bus           bus.Bus
	watcher       docker.Watcher
	templates     *template.Mapper
	stop          chan interface{}
	startListener bus.Listener
	stopListener  bus.Listener
}

// AutodiscoverBuilder builds and returns an autodiscover provider
func AutodiscoverBuilder(bus bus.Bus, c *common.Config) (autodiscover.Provider, error) {
	config := defaultConfig()
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	mapper, err := template.NewConfigMapper(config.Templates)
	if err != nil {
		return nil, err
	}

	watcher, err := docker.NewWatcher(config.Host, config.TLS)
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

	meta := common.MapStr{
		"container": common.MapStr{
			"id":     container.ID,
			"name":   container.Name,
			"image":  container.Image,
			"labels": container.Labels,
		},
	}

	// Emit container info
	d.publish(bus.Event{
		flag:     true,
		"host":   host,
		"docker": meta,
		"meta": common.MapStr{
			"docker": meta,
		},
	})

	// Emit container private ports
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
	}
	d.bus.Publish(event)
}

// Stop the autodiscover process
func (d *Provider) Stop() {
	close(d.stop)
}

func (d *Provider) String() string {
	return "docker"
}
