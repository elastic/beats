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
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
)

type pod struct {
	logger         *logp.Logger
	cleanupTimeout time.Duration
	comm           composable.DynamicProviderComm
}

type providerData struct {
	uid        string
	mapping    map[string]interface{}
	processors []map[string]interface{}
}

// NewPodWatcher creates a watcher that can discover and process pod objects
func NewPodWatcher(comm composable.DynamicProviderComm, cfg *Config, logger *logp.Logger, client k8s.Interface) (kubernetes.Watcher, error) {
	watcher, err := kubernetes.NewWatcher(client, &kubernetes.Pod{}, kubernetes.WatchOptions{
		SyncTimeout:  cfg.SyncPeriod,
		Node:         cfg.Node,
		Namespace:    cfg.Namespace,
		HonorReSyncs: true,
	}, nil)
	if err != nil {
		return nil, errors.New(err, "couldn't create kubernetes watcher")
	}
	watcher.AddEventHandler(&pod{logger, cfg.CleanupTimeout, comm})

	return watcher, nil
}

func (p *pod) emitRunning(pod *kubernetes.Pod) {
	data := generatePodData(pod)
	// Emit the pod
	// We emit Pod + containers to ensure that configs matching Pod only
	// get Pod metadata (not specific to any container)
	p.comm.AddOrUpdate(data.uid, PodPriority, data.mapping, data.processors)

	// Emit all containers in the pod
	p.emitContainers(pod, pod.Spec.Containers, pod.Status.ContainerStatuses)

	// TODO deal with init containers stopping after initialization
	p.emitContainers(pod, pod.Spec.InitContainers, pod.Status.InitContainerStatuses)
}

func (p *pod) emitContainers(pod *kubernetes.Pod, containers []kubernetes.Container, containerstatuses []kubernetes.PodContainerStatus) {

	providerDataChan := make(chan providerData)
	done := make(chan bool, 1)
	go generateContainerData(pod, containers, containerstatuses, providerDataChan, done)

	for {
		select {
		case data := <-providerDataChan:
			// Emit the container
			p.comm.AddOrUpdate(data.uid, ContainerPriority, data.mapping, data.processors)
		case <-done:
			return
		}
	}
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

func generatePodData(pod *kubernetes.Pod) providerData {
	//TODO: add metadata here too ie -> meta := s.metagen.Generate(pod)

	// Pass annotations to all events so that it can be used in templating and by annotation builders.
	annotations := common.MapStr{}
	for k, v := range pod.GetObjectMeta().GetAnnotations() {
		safemapstr.Put(annotations, k, v)
	}

	mapping := map[string]interface{}{
		"namespace": pod.GetNamespace(),
		"pod": map[string]interface{}{
			"uid":         string(pod.GetUID()),
			"name":        pod.GetName(),
			"labels":      pod.GetLabels(),
			"annotations": annotations,
			"ip":          pod.Status.PodIP,
		},
	}
	return providerData{
		uid:     string(pod.GetUID()),
		mapping: mapping,
		processors: []map[string]interface{}{
			{
				"add_fields": map[string]interface{}{
					"fields": mapping,
					"target": "kubernetes",
				},
			},
		},
	}
}

func generateContainerData(
	pod *kubernetes.Pod,
	containers []kubernetes.Container,
	containerstatuses []kubernetes.PodContainerStatus,
	dataChan chan providerData,
	done chan bool) {
	//TODO: add metadata here too ie -> meta := s.metagen.Generate()

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
		dataChan <- providerData{
			uid:        eventID,
			mapping:    mapping,
			processors: processors,
		}
	}
	done <- true
}
