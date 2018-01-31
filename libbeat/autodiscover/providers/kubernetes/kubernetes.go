package kubernetes

import (
	"time"

	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/kubernetes"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	autodiscover.ProviderRegistry.AddProvider("kubernetes", AutodiscoverBuilder)
}

// Provider implements autodiscover provider for docker containers
type Provider struct {
	config    *Config
	bus       bus.Bus
	watcher   kubernetes.Watcher
	metagen   kubernetes.MetaGenerator
	templates *template.Mapper
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

	client, err := kubernetes.GetKubernetesClient(config.InCluster, config.KubeConfig)
	if err != nil {
		return nil, err
	}

	metagen := kubernetes.NewMetaGenerator(config.IncludeAnnotations, config.IncludeLabels, config.ExcludeLabels)

	config.Host = kubernetes.DiscoverKubernetesNode(config.Host, config.InCluster, client)

	watcher, err := kubernetes.NewWatcher(client, &kubernetes.Pod{}, kubernetes.WatchOptions{
		SyncTimeout: config.SyncPeriod,
		Node:        config.Host,
		Namespace:   config.Namespace,
	})
	if err != nil {
		logp.Err("kubernetes: Couldn't create watcher for %t", &kubernetes.Pod{})
		return nil, err
	}

	p := &Provider{
		config:    config,
		bus:       bus,
		templates: mapper,
		metagen:   metagen,
		watcher:   watcher,
	}

	watcher.AddEventHandler(kubernetes.ResourceEventHandlerFuncs{
		AddFunc: func(obj kubernetes.Resource) {
			p.emit(obj.(*kubernetes.Pod), "start")
		},
		UpdateFunc: func(obj kubernetes.Resource) {
			p.emit(obj.(*kubernetes.Pod), "stop")
			p.emit(obj.(*kubernetes.Pod), "start")
		},
		DeleteFunc: func(obj kubernetes.Resource) {
			time.AfterFunc(config.CleanupTimeout, func() { p.emit(obj.(*kubernetes.Pod), "stop") })
		},
	})

	return p, nil
}

// Start for Runner interface.
func (p *Provider) Start() {
	if err := p.watcher.Start(); err != nil {
		logp.Err("Error starting kubernetes autodiscover provider: %s", err)
	}
}

func (p *Provider) emit(pod *kubernetes.Pod, flag string) {
	host := pod.Status.PodIP

	// Emit pod container IDs
	for _, c := range append(pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses...) {
		cmeta := common.MapStr{
			"id":    c.GetContainerID(),
			"name":  c.Name,
			"image": c.Image,
		}

		// Metadata appended to each event
		meta := p.metagen.ContainerMetadata(pod, c.Name)

		// Information that can be used in discovering a workload
		kubemeta := meta.Clone()
		kubemeta["container"] = cmeta

		// Emit container info
		p.publish(bus.Event{
			flag:         true,
			"host":       host,
			"kubernetes": kubemeta,
			"meta": common.MapStr{
				"kubernetes": meta,
			},
		})
	}

	// Emit pod ports
	for _, c := range pod.Spec.Containers {
		cmeta := common.MapStr{
			"name":  c.Name,
			"image": c.Image,
		}

		// Metadata appended to each event
		meta := p.metagen.ContainerMetadata(pod, c.Name)

		// Information that can be used in discovering a workload
		kubemeta := meta.Clone()
		kubemeta["container"] = cmeta

		for _, port := range c.Ports {
			event := bus.Event{
				flag:         true,
				"host":       host,
				"port":       port.ContainerPort,
				"kubernetes": kubemeta,
				"meta": common.MapStr{
					"kubernetes": meta,
				},
			}
			p.publish(event)
		}
	}
}

func (p *Provider) publish(event bus.Event) {
	// Try to match a config
	if config := p.templates.GetConfig(event); config != nil {
		event["config"] = config
	}
	p.bus.Publish(event)
}

// Stop signals the stop channel to force the watch loop routine to stop.
func (p *Provider) Stop() {
	p.watcher.Stop()
}

// String returns a description of kubernetes autodiscover provider.
func (p *Provider) String() string {
	return "kubernetes"
}
