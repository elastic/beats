// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package kubernetes

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ericchiang/k8s"
	appsv1 "github.com/ericchiang/k8s/apis/apps/v1beta1"
	"github.com/ericchiang/k8s/apis/core/v1"
	extv1 "github.com/ericchiang/k8s/apis/extensions/v1beta1"

	"github.com/elastic/beats/libbeat/logp"
)

// Max back off time for retries
const maxBackoff = 30 * time.Second

func filterByNode(node string) k8s.Option {
	return k8s.QueryParam("fieldSelector", "spec.nodeName="+node)
}

// Watcher watches Kubernetes resources events
type Watcher interface {
	// Start watching Kubernetes API for new events after resources were listed
	Start() error

	// Stop watching Kubernetes API for new events
	Stop()

	// AddEventHandler add event handlers for corresponding event type watched
	AddEventHandler(ResourceEventHandler)
}

// WatchOptions controls watch behaviors
type WatchOptions struct {
	// SyncTimeout is a timeout for listing historical resources
	SyncTimeout time.Duration
	// Node is used for filtering watched resource to given node, use "" for all nodes
	Node string
	// Namespace is used for filtering watched resource to given namespace, use "" for all namespaces
	Namespace string
}

type watcher struct {
	client              *k8s.Client
	options             WatchOptions
	lastResourceVersion string
	ctx                 context.Context
	stop                context.CancelFunc
	resourceList        k8s.ResourceList
	k8sResourceFactory  func() k8s.Resource
	items               func() []k8s.Resource
	handler             ResourceEventHandler
	logger              *logp.Logger
}

// NewWatcher initializes the watcher client to provide a events handler for
// resource from the cluster (filtered to the given node)
func NewWatcher(client *k8s.Client, resource Resource, options WatchOptions) (Watcher, error) {
	ctx, cancel := context.WithCancel(context.Background())
	w := &watcher{
		client:              client,
		options:             options,
		lastResourceVersion: "0",
		ctx:                 ctx,
		stop:                cancel,
		logger:              logp.NewLogger("kubernetes"),
	}
	switch resource.(type) {
	// add resource type which you want to support watching here
	// note that you might need add Register like event in types.go init func
	// if types were not registered by k8s library
	// k8s.Register("", "v1", "events", true, &v1.Event{})
	// k8s.RegisterList("", "v1", "events", true, &v1.EventList{})
	case *Pod:
		list := &v1.PodList{}
		w.resourceList = list
		w.k8sResourceFactory = func() k8s.Resource { return &v1.Pod{} }
		w.items = func() []k8s.Resource {
			rs := make([]k8s.Resource, 0, len(list.Items))
			for _, item := range list.Items {
				rs = append(rs, item)
			}
			return rs
		}
	case *Event:
		list := &v1.EventList{}
		w.resourceList = list
		w.k8sResourceFactory = func() k8s.Resource { return &v1.Event{} }
		w.items = func() []k8s.Resource {
			rs := make([]k8s.Resource, 0, len(list.Items))
			for _, item := range list.Items {
				rs = append(rs, item)
			}
			return rs
		}
	case *Node:
		list := &v1.NodeList{}
		w.resourceList = list
		w.k8sResourceFactory = func() k8s.Resource { return &v1.Node{} }
		w.items = func() []k8s.Resource {
			rs := make([]k8s.Resource, 0, len(list.Items))
			for _, item := range list.Items {
				rs = append(rs, item)
			}
			return rs
		}
	case *Deployment:
		list := &appsv1.DeploymentList{}
		w.resourceList = list
		w.k8sResourceFactory = func() k8s.Resource { return &appsv1.Deployment{} }
		w.items = func() []k8s.Resource {
			rs := make([]k8s.Resource, 0, len(list.Items))
			for _, item := range list.Items {
				rs = append(rs, item)
			}
			return rs
		}
	case *ReplicaSet:
		list := &extv1.ReplicaSetList{}
		w.resourceList = list
		w.k8sResourceFactory = func() k8s.Resource { return &extv1.ReplicaSet{} }
		w.items = func() []k8s.Resource {
			rs := make([]k8s.Resource, 0, len(list.Items))
			for _, item := range list.Items {
				rs = append(rs, item)
			}
			return rs
		}
	case *StatefulSet:
		list := &appsv1.StatefulSetList{}
		w.resourceList = list
		w.k8sResourceFactory = func() k8s.Resource { return &appsv1.StatefulSet{} }
		w.items = func() []k8s.Resource {
			rs := make([]k8s.Resource, 0, len(list.Items))
			for _, item := range list.Items {
				rs = append(rs, item)
			}
			return rs
		}
	default:
		return nil, fmt.Errorf("unsupported resource type for watching %T", resource)
	}
	return w, nil
}

func (w *watcher) AddEventHandler(h ResourceEventHandler) {
	w.handler = h
}

func (w *watcher) buildOpts() []k8s.Option {
	options := []k8s.Option{k8s.ResourceVersion(w.lastResourceVersion)}
	if w.options.Node != "" {
		options = append(options, filterByNode(w.options.Node))
	}
	return options
}

func (w *watcher) sync() error {
	ctx, cancel := context.WithTimeout(w.ctx, w.options.SyncTimeout)
	defer cancel()

	logp.Info("kubernetes: Performing a resource sync for %T", w.resourceList)
	err := w.client.List(ctx, w.options.Namespace, w.resourceList, w.buildOpts()...)
	if err != nil {
		logp.Err("kubernetes: Performing a resource sync err %s for %T", err.Error(), w.resourceList)
		return err
	}

	w.logger.Debugf("Got %v items from the resource sync", len(w.items()))
	for _, item := range w.items() {
		w.onAdd(item)
	}

	w.logger.Debugf("Done syncing %v items from the resource sync", len(w.items()))
	// Store last version
	w.lastResourceVersion = w.resourceList.GetMetadata().GetResourceVersion()

	logp.Info("kubernetes: %s", "Resource sync done")
	return nil
}

func (w *watcher) onAdd(obj Resource) {
	w.handler.OnAdd(obj)
}

func (w *watcher) onUpdate(obj Resource) {
	w.handler.OnUpdate(obj)
}

func (w *watcher) onDelete(obj Resource) {
	w.handler.OnDelete(obj)
}

// Start watching pods
func (w *watcher) Start() error {
	// Make sure that events don't flow into the annotator before informer is fully set up
	// Sync initial state:
	err := w.sync()
	if err != nil {
		w.Stop()
		return err
	}

	// Watch for new changes
	go w.watch()

	return nil
}

func (w *watcher) watch() {
	// Failures counter, do exponential backoff on retries
	var failures uint

	for {
		select {
		case <-w.ctx.Done():
			logp.Info("kubernetes: %s", "Watching API for resource events stopped")
			return
		default:
		}

		logp.Info("kubernetes: %s", "Watching API for resource events")

		watcher, err := w.client.Watch(w.ctx, w.options.Namespace, w.k8sResourceFactory(), w.buildOpts()...)
		if err != nil {
			//watch failures should be logged and gracefully failed over as metadata retrieval
			//should never stop.
			logp.Err("kubernetes: Watching API error %v", err)
			backoff(failures)
			failures++
			continue
		}

		for {
			r := w.k8sResourceFactory()
			eventType, err := watcher.Next(r)
			if err != nil {
				watcher.Close()
				switch err {
				case io.EOF:
					logp.Debug("kubernetes", "EOF while watching API")
				case io.ErrUnexpectedEOF:
					logp.Info("kubernetes: Unexpected EOF while watching API")
				default:
					// This is an error event which can be recovered by moving to the latest resource version
					logp.Err("kubernetes: Watching API error %v, ignoring event and moving to most recent resource version", err)
					w.lastResourceVersion = ""

				}
				break
			}
			failures = 0
			switch eventType {
			case k8s.EventAdded:
				w.onAdd(r)
			case k8s.EventModified:
				w.onUpdate(r)
			case k8s.EventDeleted:
				w.onDelete(r)
			default:
				logp.Err("kubernetes: Watching API error with event type %s", eventType)
			}
		}
	}
}

func (w *watcher) Stop() {
	w.stop()
}
func backoff(failures uint) {
	wait := 1 << failures * time.Second
	if wait > maxBackoff {
		wait = maxBackoff
	}
	time.Sleep(wait)
}
