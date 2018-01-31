package kubernetes

import (
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
	config         *Config
	bus            bus.Bus
	watcher        kubernetes.Watcher
	metagen        kubernetes.MetaGenerator
	templates      *template.Mapper
	stop           chan interface{}
	startListener  bus.Listener
	stopListener   bus.Listener
	updateListener bus.Listener
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

	config.Host = kubernetes.DiscoverKubernetesNode(config.Host, client)
	watcher := kubernetes.NewWatcher(client.CoreV1(), config.SyncPeriod, config.CleanupTimeout, config.Host)

	start := watcher.ListenStart()
	stop := watcher.ListenStop()
	update := watcher.ListenUpdate()

	if err := watcher.Start(); err != nil {
		return nil, err
	}

	return &Provider{
		config:         config,
		bus:            bus,
		templates:      mapper,
		metagen:        metagen,
		watcher:        watcher,
		stop:           make(chan interface{}),
		startListener:  start,
		stopListener:   stop,
		updateListener: update,
	}, nil
}

// Start the autodiscover provider. Start and stop listeners work the
// conventional way. Update listener triggers a stop and then a start
// to simulate an update.
func (p *Provider) Start() {
	go func() {
		for {
			select {
			case <-p.stop:
				p.startListener.Stop()
				p.stopListener.Stop()
				return

			case event := <-p.startListener.Events():
				p.emit(event, "start")

			case event := <-p.stopListener.Events():
				p.emit(event, "stop")

			case event := <-p.updateListener.Events():
				//On updates, first send a stop signal followed by a start signal to simulate a restart
				p.emit(event, "stop")
				p.emit(event, "start")
			}
		}
	}()
}

func (p *Provider) emit(event bus.Event, flag string) {
	pod, ok := event["pod"].(*kubernetes.Pod)
	if !ok {
		logp.Err("Couldn't get a pod from watcher event")
		return
	}

	host := pod.Status.PodIP
	containerIDs := map[string]string{}

	// Emit pod container IDs
	for _, c := range append(pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses...) {
		cid := c.GetContainerID()
		containerIDs[c.Name] = cid

		cmeta := common.MapStr{
			"id":    cid,
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
			"id":    containerIDs[c.Name],
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
	close(p.stop)
}

// String returns a description of kubernetes autodiscover provider.
func (p *Provider) String() string {
	return "kubernetes"
}
