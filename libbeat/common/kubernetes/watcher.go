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
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/elastic/beats/v7/libbeat/logp"
)

const (
	add    = "add"
	update = "update"
	delete = "delete"
)

var (
	accessor = meta.NewAccessor()
)

// Watcher watches Kubernetes resources events
type Watcher interface {
	// Start watching Kubernetes API for new events after resources were listed
	Start() error

	// Stop watching Kubernetes API for new events
	Stop()

	// AddEventHandler add event handlers for corresponding event type watched
	AddEventHandler(ResourceEventHandler)

	// Store returns the store object for the watcher
	Store() cache.Store
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

type item struct {
	object interface{}
	state  string
}

type watcher struct {
	client   kubernetes.Interface
	informer cache.SharedInformer
	store    cache.Store
	queue    workqueue.RateLimitingInterface
	ctx      context.Context
	stop     context.CancelFunc
	handler  ResourceEventHandler
	logger   *logp.Logger
}

// NewWatcher initializes the watcher client to provide a events handler for
// resource from the cluster (filtered to the given node)
func NewWatcher(client kubernetes.Interface, resource Resource, opts WatchOptions, indexers cache.Indexers) (Watcher, error) {
	var store cache.Store
	var queue workqueue.RateLimitingInterface

	informer, objType, err := NewInformer(client, resource, opts, indexers)
	if err != nil {
		return nil, err
	}

	store = informer.GetStore()
	queue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), objType)
	ctx, cancel := context.WithCancel(context.Background())

	w := &watcher{
		client:   client,
		informer: informer,
		store:    store,
		queue:    queue,
		ctx:      ctx,
		stop:     cancel,
		logger:   logp.NewLogger("kubernetes"),
		handler:  NoOpEventHandlerFuncs{},
	}

	w.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(o interface{}) {
			w.enqueue(o, add)
		},
		DeleteFunc: func(o interface{}) {
			w.enqueue(o, delete)
		},
		UpdateFunc: func(o, n interface{}) {
			old, _ := accessor.ResourceVersion(o.(runtime.Object))
			new, _ := accessor.ResourceVersion(n.(runtime.Object))

			// Only enqueue changes that have a different resource versions to avoid processing resyncs.
			if old != new {
				w.enqueue(n, update)
			}
		},
	})

	return w, nil
}

// AddEventHandler adds a resource handler to process each request that is coming into the watcher
func (w *watcher) AddEventHandler(h ResourceEventHandler) {
	w.handler = h
}

// Store returns the store object for the resource that is being watched
func (w *watcher) Store() cache.Store {
	return w.store
}

// Start watching pods
func (w *watcher) Start() error {
	go w.informer.Run(w.ctx.Done())

	if !cache.WaitForCacheSync(w.ctx.Done(), w.informer.HasSynced) {
		return fmt.Errorf("kubernetes informer unable to sync cache")
	}

	w.logger.Debugf("cache sync done")

	//TODO: Do we run parallel workers for this? It is useful when we run metricbeat as one instance per cluster?

	// Wrap the process function with wait.Until so that if the controller crashes, it starts up again after a second.
	go wait.Until(func() {
		for w.process(w.ctx) {
		}
	}, time.Second*1, w.ctx.Done())

	return nil
}

func (w *watcher) Stop() {
	w.queue.ShutDown()
	w.stop()
}

// enqueue takes the most recent object that was received, figures out the namespace/name of the object
// and adds it to the work queue for processing.
func (w *watcher) enqueue(obj interface{}, state string) {
	// DeletionHandlingMetaNamespaceKeyFunc that we get a key only if the resource's state is not Unknown.
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}

	w.queue.Add(&item{key, state})
}

// process gets the top of the work queue and processes the object that is received.
func (w *watcher) process(ctx context.Context) bool {
	keyObj, quit := w.queue.Get()
	if quit {
		return false
	}

	err := func(obj interface{}) error {
		defer w.queue.Done(obj)

		var entry *item
		var ok bool
		if entry, ok = obj.(*item); !ok {
			w.queue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected *item in workqueue but got %#v", obj))
			return nil
		}

		key := entry.object.(string)

		o, exists, err := w.store.GetByKey(key)
		if err != nil {
			return nil
		}
		if !exists {
			return nil
		}

		switch entry.state {
		case add:
			w.handler.OnAdd(o)
		case update:
			w.handler.OnUpdate(o)
		case delete:
			w.handler.OnDelete(o)
		}

		return nil
	}(keyObj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}
