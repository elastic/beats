// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kubernetes

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/safemapstr"

	k8s "k8s.io/client-go/kubernetes"

	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes/metadata"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
)

type pod struct {
	logger         *logp.Logger
	cleanupTimeout time.Duration
	comm           composable.DynamicProviderComm
	scope          string
	config         *Config
	metagen        metadata.MetaGen
}

type providerData struct {
	uid        string
	mapping    map[string]interface{}
	processors []map[string]interface{}
}

// NewPodWatcher creates a watcher that can discover and process pod objects
func NewPodWatcher(
	comm composable.DynamicProviderComm,
	cfg *Config,
	logger *logp.Logger,
	client k8s.Interface,
	scope string,
	rawConfig *common.Config) (kubernetes.Watcher, error) {
	watcher, err := kubernetes.NewWatcher(client, &kubernetes.Pod{}, kubernetes.WatchOptions{
		SyncTimeout:  cfg.SyncPeriod,
		Node:         cfg.Node,
		Namespace:    cfg.Namespace,
		HonorReSyncs: true,
	}, nil)
	if err != nil {
		return nil, errors.New(err, "couldn't create kubernetes watcher")
	}

	options := kubernetes.WatchOptions{
		SyncTimeout: cfg.SyncPeriod,
		Node:        cfg.Node,
	}
	metaConf := cfg.AddResourceMetadata
	if metaConf == nil {
		metaConf = metadata.GetDefaultResourceMetadataConfig()
	}
	nodeWatcher, err := kubernetes.NewWatcher(client, &kubernetes.Node{}, options, nil)
	if err != nil {
		logger.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Node{}, err)
	}
	namespaceWatcher, err := kubernetes.NewWatcher(client, &kubernetes.Namespace{}, kubernetes.WatchOptions{
		SyncTimeout: cfg.SyncPeriod,
	}, nil)
	if err != nil {
		logger.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Namespace{}, err)
	}

	metaGen := metadata.GetPodMetaGen(rawConfig, watcher, nodeWatcher, namespaceWatcher, metaConf)
	watcher.AddEventHandler(&pod{
		logger,
		cfg.CleanupTimeout,
		comm,
		scope,
		cfg,
		metaGen,
	})

	return watcher, nil
}

func (p *pod) emitRunning(pod *kubernetes.Pod) {

	data := generatePodData(pod, p.config, p.metagen)
	data.mapping["scope"] = p.scope
	// Emit the pod
	// We emit Pod + containers to ensure that configs matching Pod only
	// get Pod metadata (not specific to any container)
	p.comm.AddOrUpdate(data.uid, PodPriority, data.mapping, data.processors)

	// Emit all containers in the pod
	p.emitContainers(pod, pod.Spec.Containers, pod.Status.ContainerStatuses)

	// TODO: deal with init containers stopping after initialization
	p.emitContainers(pod, pod.Spec.InitContainers, pod.Status.InitContainerStatuses)

	// Get ephemeral containers and their status
	ephContainers, ephContainersStatuses := getEphemeralContainers(pod)
	p.emitContainers(pod, ephContainers, ephContainersStatuses)

}

func (p *pod) emitContainers(
	pod *kubernetes.Pod,
	containers []kubernetes.Container,
	containerstatuses []kubernetes.PodContainerStatus) {
	generateContainerData(p.comm, pod, containers, containerstatuses, p.config, p.metagen)
}

func (p *pod) emitStopped(pod *kubernetes.Pod) {
	p.comm.Remove(string(pod.GetUID()))

	for _, c := range pod.Spec.Containers {
		// ID is the combination of pod UID + container name
		eventID := fmt.Sprintf("%s.%s", pod.GetObjectMeta().GetUID(), c.Name)
		p.comm.Remove(eventID)
	}

	for _, c := range pod.Spec.InitContainers {
		// ID is the combination of pod UID + container name
		eventID := fmt.Sprintf("%s.%s", pod.GetObjectMeta().GetUID(), c.Name)
		p.comm.Remove(eventID)
	}
}

// OnAdd ensures processing of pod objects that are newly added
func (p *pod) OnAdd(obj interface{}) {
	p.logger.Debugf("pod add: %+v", obj)
	p.emitRunning(obj.(*kubernetes.Pod))
}

// OnUpdate emits events for a given pod depending on the state of the pod,
// if it is terminating, a stop event is scheduled, if not, a stop and a start
// events are sent sequentially to recreate the resources assotiated to the pod.
func (p *pod) OnUpdate(obj interface{}) {
	pod := obj.(*kubernetes.Pod)

	p.logger.Debugf("pod update for pod: %+v, status: %+v", pod.Name, pod.Status.Phase)
	switch pod.Status.Phase {
	case kubernetes.PodSucceeded, kubernetes.PodFailed:
		time.AfterFunc(p.cleanupTimeout, func() { p.emitStopped(pod) })
		return
	case kubernetes.PodPending:
		p.logger.Debugf("pod update (pending): don't know what to do with this pod yet, skipping for now: %+v", obj)
		return
	}

	p.logger.Debugf("pod update: %+v", obj)
	p.emitRunning(pod)
}

// OnDelete stops pod objects that are deleted
func (p *pod) OnDelete(obj interface{}) {
	p.logger.Debugf("pod delete: %+v", obj)
	pod := obj.(*kubernetes.Pod)
	time.AfterFunc(p.cleanupTimeout, func() { p.emitStopped(pod) })
}

func generatePodData(pod *kubernetes.Pod, cfg *Config, kubeMetaGen metadata.MetaGen) providerData {

	meta := kubeMetaGen.Generate(pod)
	kubemetaMap, err := meta.GetValue("kubernetes")
	if err != nil {
		return providerData{}
	}

	// k8sMapping includes only the metadata that fall under kubernetes.*
	// and these are available as dynamic vars through the provider
	k8sMapping := map[string]interface{}(kubemetaMap.(common.MapStr))

	processors := []map[string]interface{}{}
	// meta map includes metadata that go under kubernetes.*
	// but also other ECS fields like orchestrator.*
	for field, metaMap := range meta {
		processor := map[string]interface{}{
			"add_fields": map[string]interface{}{
				"fields": metaMap,
				"target": field,
			},
		}
		processors = append(processors, processor)
	}

	return providerData{
		uid:        string(pod.GetUID()),
		mapping:    k8sMapping,
		processors: processors,
	}
}

func generateContainerData(
	comm composable.DynamicProviderComm,
	pod *kubernetes.Pod,
	containers []kubernetes.Container,
	containerstatuses []kubernetes.PodContainerStatus,
	cfg *Config,
	kubeMetaGen metadata.MetaGen) {

	containerIDs := map[string]string{}
	runtimes := map[string]string{}
	for _, c := range containerstatuses {
		cid, runtime := kubernetes.ContainerIDWithRuntime(c)
		containerIDs[c.Name] = cid
		runtimes[c.Name] = runtime
	}

	// Pass annotations to all events so that it can be used in templating and by annotation builders.
	annotations := common.MapStr{}
	for k, v := range pod.GetObjectMeta().GetAnnotations() {
		safemapstr.Put(annotations, k, v)
	}

	for _, c := range containers {
		// If it doesn't have an ID, container doesn't exist in
		// the runtime, emit only an event if we are stopping, so
		// we are sure of cleaning up configurations.
		cid := containerIDs[c.Name]
		if cid == "" {
			continue
		}

		// ID is the combination of pod UID + container name
		eventID := fmt.Sprintf("%s.%s", pod.GetObjectMeta().GetUID(), c.Name)

		meta := kubeMetaGen.Generate(pod, metadata.WithFields("container.name", c.Name))
		kubemetaMap, err := meta.GetValue("kubernetes")
		if err != nil {
			continue
		}

		// k8sMapping includes only the metadata that fall under kubernetes.*
		// and these are available as dynamic vars through the provider
		k8sMapping := map[string]interface{}(kubemetaMap.(common.MapStr))

		// add annotations to be discoverable by templates
		k8sMapping["annotations"] = annotations

		//container ECS fields
		cmeta := common.MapStr{
			"id":      cid,
			"runtime": runtimes[c.Name],
			"image": common.MapStr{
				"name": c.Image,
			},
		}

		processors := []map[string]interface{}{
			{
				"add_fields": map[string]interface{}{
					"fields": cmeta,
					"target": "container",
				},
			},
		}
		// meta map includes metadata that go under kubernetes.*
		// but also other ECS fields like orchestrator.*
		for field, metaMap := range meta {
			processor := map[string]interface{}{
				"add_fields": map[string]interface{}{
					"fields": metaMap,
					"target": field,
				},
			}
			processors = append(processors, processor)
		}

		// add container metadata under kubernetes.container.* to
		// make them available to dynamic var resolution
		containerMeta := common.MapStr{
			"id":      cid,
			"name":    c.Name,
			"image":   c.Image,
			"runtime": runtimes[c.Name],
		}
		if len(c.Ports) > 0 {
			for _, port := range c.Ports {
				containerMeta.Put("port", port.ContainerPort)
				containerMeta.Put("port_name", port.Name)
				k8sMapping["container"] = containerMeta
				comm.AddOrUpdate(eventID, ContainerPriority, k8sMapping, processors)
			}
		} else {
			k8sMapping["container"] = containerMeta
			comm.AddOrUpdate(eventID, ContainerPriority, k8sMapping, processors)
		}
	}
}

func getEphemeralContainers(pod *kubernetes.Pod) ([]kubernetes.Container, []kubernetes.PodContainerStatus) {
	var ephContainers []kubernetes.Container
	var ephContainersStatuses []kubernetes.PodContainerStatus
	for _, c := range pod.Spec.EphemeralContainers {
		c := kubernetes.Container(c.EphemeralContainerCommon)
		ephContainers = append(ephContainers, c)
	}
	for _, s := range pod.Status.EphemeralContainerStatuses {
		ephContainersStatuses = append(ephContainersStatuses, s)
	}
	return ephContainers, ephContainersStatuses
}
