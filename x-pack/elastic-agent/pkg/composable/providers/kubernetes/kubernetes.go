// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kubernetes

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

const (
	// PodPriority is the priority that pod mappings are added to the provider.
	PodPriority = 0
	// ContainerPriority is the priority that container mappings are added to the provider.
	ContainerPriority = 1
)

func init() {
	composable.Providers.AddDynamicProvider("kubernetes", DynamicProviderBuilder)
}

type dynamicProvider struct {
	logger *logger.Logger
	config *Config
}

type eventWatcher struct {
	logger         *logger.Logger
	cleanupTimeout time.Duration
	comm           composable.DynamicProviderComm
}

// DynamicProviderBuilder builds the dynamic provider.
func DynamicProviderBuilder(logger *logger.Logger, c *config.Config) (composable.DynamicProvider, error) {
	var cfg Config
	if c == nil {
		c = config.New()
	}
	err := c.Unpack(&cfg)
	if err != nil {
		return nil, errors.New(err, "failed to unpack configuration")
	}
	return &dynamicProvider{logger, &cfg}, nil
}

// Run runs the environment context provider.
func (p *dynamicProvider) Run(comm composable.DynamicProviderComm) error {
	client, err := kubernetes.GetKubernetesClient(p.config.KubeConfig)
	if err != nil {
		// info only; return nil (do nothing)
		p.logger.Debugf("Kubernetes provider skipped, unable to connect: %s", err)
		return nil
	}

	// Ensure that node is set correctly whenever the scope is set to "node". Make sure that node is empty
	// when cluster scope is enforced.
	p.logger.Infof("Kubernetes provider started with %s scope", p.config.Scope)
	if p.config.Scope == "node" {
		p.logger.Debugf("Initializing Kubernetes watcher using node: %v", p.config.Node)
		p.config.Node = kubernetes.DiscoverKubernetesNode(p.logger, p.config.Node, kubernetes.IsInCluster(p.config.KubeConfig), client)
	} else {
		p.config.Node = ""
	}

	watcher, err := kubernetes.NewWatcher(client, &kubernetes.Pod{}, kubernetes.WatchOptions{
		SyncTimeout: p.config.SyncPeriod,
		Node:        p.config.Node,
		//Namespace:   p.config.Namespace,
	}, nil)
	if err != nil {
		return errors.New(err, "couldn't create kubernetes watcher")
	}
	watcher.AddEventHandler(&eventWatcher{p.logger, p.config.CleanupTimeout, comm})

	err = watcher.Start()
	if err != nil {
		return errors.New(err, "couldn't start kubernetes watcher")
	}

	return nil
}

func (p *eventWatcher) emitRunning(pod *kubernetes.Pod) {
	mapping := map[string]interface{}{
		"namespace": pod.GetNamespace(),
		"pod": map[string]interface{}{
			"uid":    string(pod.GetUID()),
			"name":   pod.GetName(),
			"labels": pod.GetLabels(),
			"ip":     pod.Status.PodIP,
		},
	}

	processors := []map[string]interface{}{
		{
			"add_fields": map[string]interface{}{
				"fields": mapping,
				"target": "kubernetes",
			},
		},
	}

	// Emit the pod
	// We emit Pod + containers to ensure that configs matching Pod only
	// get Pod metadata (not specific to any container)
	p.comm.AddOrUpdate(string(pod.GetUID()), PodPriority, mapping, processors)

	// Emit all containers in the pod
	p.emitContainers(pod, pod.Spec.Containers, pod.Status.ContainerStatuses)

	// TODO deal with init containers stopping after initialization
	p.emitContainers(pod, pod.Spec.InitContainers, pod.Status.InitContainerStatuses)
}

func (p *eventWatcher) emitContainers(pod *kubernetes.Pod, containers []kubernetes.Container, containerstatuses []kubernetes.PodContainerStatus) {
	// Collect all runtimes from status information.
	containerIDs := map[string]string{}
	runtimes := map[string]string{}
	for _, c := range containerstatuses {
		cid, runtime := kubernetes.ContainerIDWithRuntime(c)
		containerIDs[c.Name] = cid
		runtimes[c.Name] = runtime
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

		mapping := map[string]interface{}{
			"namespace": pod.GetNamespace(),
			"pod": map[string]interface{}{
				"uid":    string(pod.GetUID()),
				"name":   pod.GetName(),
				"labels": pod.GetLabels(),
				"ip":     pod.Status.PodIP,
			},
			"container": map[string]interface{}{
				"id":      cid,
				"name":    c.Name,
				"image":   c.Image,
				"runtime": runtimes[c.Name],
			},
		}

		processors := []map[string]interface{}{
			{
				"add_fields": map[string]interface{}{
					"fields": mapping,
					"target": "kubernetes",
				},
			},
		}

		// Emit the container
		p.comm.AddOrUpdate(eventID, ContainerPriority, mapping, processors)
	}
}

func (p *eventWatcher) emitStopped(pod *kubernetes.Pod) {
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
func (p *eventWatcher) OnAdd(obj interface{}) {
	p.logger.Debugf("pod add: %+v", obj)
	p.emitRunning(obj.(*kubernetes.Pod))
}

// OnUpdate emits events for a given pod depending on the state of the pod,
// if it is terminating, a stop event is scheduled, if not, a stop and a start
// events are sent sequentially to recreate the resources assotiated to the pod.
func (p *eventWatcher) OnUpdate(obj interface{}) {
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
func (p *eventWatcher) OnDelete(obj interface{}) {
	p.logger.Debugf("pod delete: %+v", obj)
	pod := obj.(*kubernetes.Pod)
	time.AfterFunc(p.cleanupTimeout, func() { p.emitStopped(pod) })
}
