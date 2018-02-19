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
	autodiscover.Registry.AddProvider("kubernetes", AutodiscoverBuilder)
}

// Provider implements autodiscover provider for docker containers
type Provider struct {
	config    *Config
	bus       bus.Bus
	watcher   kubernetes.Watcher
	metagen   kubernetes.MetaGenerator
	templates *template.Mapper
	builders  autodiscover.Builders
}

// AutodiscoverBuilder builds and returns an autodiscover provider
func AutodiscoverBuilder(bus bus.Bus, mapper *template.Mapper, builders autodiscover.Builders, c *common.Config) (autodiscover.Provider, error) {
	config := defaultConfig()
	err := c.Unpack(&config)
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
		builders:  builders,
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
	// Emit events for all containers
	p.emitEvents(pod, flag, pod.Spec.Containers, pod.Status.ContainerStatuses)

	// Emit events for all initContainers
	p.emitEvents(pod, flag, pod.Spec.InitContainers, pod.Status.InitContainerStatuses)
}

func (p *Provider) emitEvents(pod *kubernetes.Pod, flag string, containers []kubernetes.Container,
	containerstatuses []kubernetes.PodContainerStatus) {
	host := pod.Status.PodIP

	// Collect all container IDs from status information
	containerIDs := map[string]string{}
	for _, c := range containerstatuses {
		cid := c.GetContainerID()
		containerIDs[c.Name] = cid
	}

	// Emit container and port information
	for _, c := range containers {
		cmeta := common.MapStr{
			"id":    containerIDs[c.Name],
			"name":  c.Name,
			"image": c.Image,
		}

		// Metadata appended to each event
		meta := p.metagen.ContainerMetadata(pod, c.Name)

		// Information that can be used in discovering a workload
		kubemeta := meta.Clone()
		kubemeta["container"] = cmeta

		// Pass annotations to all events so that it can be used in templating and by annotation builders.
		kubemeta["annotations"] = pod.GetMetadata().Annotations

		// Without this check there would be overlapping configurations with and without ports.
		if len(c.Ports) == 0 {
			event := bus.Event{
				flag:         true,
				"host":       host,
				"kubernetes": kubemeta,
				"meta": common.MapStr{
					"kubernetes": meta,
				},
			}
			p.publish(event)
		}

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
	} else {
		// Try to build a config with enabled builders. Send a provider agnostic payload.
		// Builders are Beat specific.
		e := bus.Event{}
		kubeMeta, _ := event["kubernetes"].(common.MapStr)
		if host, ok := event["host"]; ok {
			e["host"] = host
		}
		if port, ok := event["port"]; ok {
			e["port"] = port
		}
		e["annotations"] = kubeMeta["annotations"]
		e["container"] = kubeMeta["container"]
		if config := p.builders.GetConfig(e); config != nil {
			event["config"] = config
		}
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
