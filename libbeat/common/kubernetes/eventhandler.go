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
	"sync"
)

// ResourceEventHandler can handle notifications for events that happen to a
// resource. The events are informational only, so you can't return an
// error.
//  * OnAdd is called when an object is added.
//  * OnUpdate is called when an object is modified. Note that oldObj is the
//      last known state of the object-- it is possible that several changes
//      were combined together, so you can't use this to see every single
//      change. OnUpdate is also called when a re-list happens, and it will
//      get called even if nothing changed. This is useful for periodically
//      evaluating or syncing something.
//  * OnDelete will get the final state of the item if it is known, otherwise
//      it will get an object of type DeletedFinalStateUnknown. This can
//      happen if the watch is closed and misses the delete event and we don't
//      notice the deletion until the subsequent re-list.
// TODO: allow the On* methods to return an error so that the RateLimited WorkQueue
// TODO: can requeue the failed event processing.
type ResourceEventHandler interface {
	OnAdd(obj interface{})
	OnUpdate(obj interface{})
	OnDelete(obj interface{})
}

// ResourceEventHandlerFuncs is an adaptor to let you easily specify as many or
// as few of the notification functions as you want while still implementing
// ResourceEventHandler.
type ResourceEventHandlerFuncs struct {
	AddFunc    func(obj interface{})
	UpdateFunc func(obj interface{})
	DeleteFunc func(obj interface{})
}

// OnAdd calls AddFunc if it's not nil.
func (r ResourceEventHandlerFuncs) OnAdd(obj interface{}) {
	if r.AddFunc != nil {
		r.AddFunc(obj)
	}
}

// OnUpdate calls UpdateFunc if it's not nil.
func (r ResourceEventHandlerFuncs) OnUpdate(obj interface{}) {
	if r.UpdateFunc != nil {
		r.UpdateFunc(obj)
	}
}

// OnDelete calls DeleteFunc if it's not nil.
func (r ResourceEventHandlerFuncs) OnDelete(obj interface{}) {
	if r.DeleteFunc != nil {
		r.DeleteFunc(obj)
	}
}

// NoOpEventHandlerFuncs ensures that watcher reconciliation can happen even without the required funcs
type NoOpEventHandlerFuncs struct {
}

// OnAdd does a no-op on an add event
func (n NoOpEventHandlerFuncs) OnAdd(obj interface{}) {

}

// OnUpdate does a no-op on an update event
func (n NoOpEventHandlerFuncs) OnUpdate(obj interface{}) {

}

// OnDelete does a no-op on a delete event
func (n NoOpEventHandlerFuncs) OnDelete(obj interface{}) {

}

// FilteringResourceEventHandler applies the provided filter to all events coming
// in, ensuring the appropriate nested handler method is invoked. An object
// that starts passing the filter after an update is considered an add, and an
// object that stops passing the filter after an update is considered a delete.
type FilteringResourceEventHandler struct {
	FilterFunc func(obj interface{}) bool
	Handler    ResourceEventHandler
}

// OnAdd calls the nested handler only if the filter succeeds
func (r FilteringResourceEventHandler) OnAdd(obj interface{}) {
	if !r.FilterFunc(obj) {
		return
	}
	r.Handler.OnAdd(obj)
}

// OnUpdate ensures the proper handler is called depending on whether the filter matches
func (r FilteringResourceEventHandler) OnUpdate(obj interface{}) {
	if !r.FilterFunc(obj) {
		return
	}
	r.Handler.OnUpdate(obj)
}

// OnDelete calls the nested handler only if the filter succeeds
func (r FilteringResourceEventHandler) OnDelete(obj interface{}) {
	if !r.FilterFunc(obj) {
		return
	}
	r.Handler.OnDelete(obj)
}

// podUpdaterHandlerFunc is a function that handles pod updater notifications.
type podUpdaterHandlerFunc func(interface{})

// podUpdaterStore is the interface that an object needs to implement to be
// used as a pod updater store.
type podUpdaterStore interface {
	List() []interface{}
}

// namespacePodUpdater notifies updates on pods when their namespaces are updated.
type namespacePodUpdater struct {
	handler podUpdaterHandlerFunc
	store   podUpdaterStore
	locker  sync.Locker
}

// NewNamespacePodUpdater creates a namespacePodUpdater
func NewNamespacePodUpdater(handler podUpdaterHandlerFunc, store podUpdaterStore, locker sync.Locker) *namespacePodUpdater {
	return &namespacePodUpdater{
		handler: handler,
		store:   store,
		locker:  locker,
	}
}

// OnUpdate handles update events on namespaces.
func (n *namespacePodUpdater) OnUpdate(obj interface{}) {
	ns, ok := obj.(*Namespace)
	if !ok {
		return
	}

	// n.store.List() returns a snapshot at this point. If a delete is received
	// from the main watcher, this loop may generate an update event after the
	// delete is processed, leaving configurations that would never be deleted.
	// Also this loop can miss updates, what could leave outdated configurations.
	// Avoid these issues by locking the processing of events from the main watcher.
	if n.locker != nil {
		n.locker.Lock()
		defer n.locker.Unlock()
	}
	for _, pod := range n.store.List() {
		pod, ok := pod.(*Pod)
		if ok && pod.Namespace == ns.Name {
			n.handler(pod)
		}
	}
}

// OnAdd handles add events on namespaces. Nothing to do, if pods are added to this
// namespace they will generate their own add events.
func (*namespacePodUpdater) OnAdd(interface{}) {}

// OnDelete handles delete events on namespaces. Nothing to do, if pods are deleted from this
// namespace they will generate their own delete events.
func (*namespacePodUpdater) OnDelete(interface{}) {}

// nodePodUpdater notifies updates on pods when their nodes are updated.
type nodePodUpdater struct {
	handler podUpdaterHandlerFunc
	store   podUpdaterStore
	locker  sync.Locker
}

// NewNodePodUpdater creates a nodePodUpdater
func NewNodePodUpdater(handler podUpdaterHandlerFunc, store podUpdaterStore, locker sync.Locker) *nodePodUpdater {
	return &nodePodUpdater{
		handler: handler,
		store:   store,
		locker:  locker,
	}
}

// OnUpdate handles update events on nodes.
func (n *nodePodUpdater) OnUpdate(obj interface{}) {
	node, ok := obj.(*Node)
	if !ok {
		return
	}

	// n.store.List() returns a snapshot at this point. If a delete is received
	// from the main watcher, this loop may generate an update event after the
	// delete is processed, leaving configurations that would never be deleted.
	// Also this loop can miss updates, what could leave outdated configurations.
	// Avoid these issues by locking the processing of events from the main watcher.
	if n.locker != nil {
		n.locker.Lock()
		defer n.locker.Unlock()
	}
	for _, pod := range n.store.List() {
		pod, ok := pod.(*Pod)
		if ok && pod.Spec.NodeName == node.Name {
			n.handler(pod)
		}
	}
}

// OnAdd handles add events on namespaces. Nothing to do, if pods are added to this
// namespace they will generate their own add events.
func (*nodePodUpdater) OnAdd(interface{}) {}

// OnDelete handles delete events on namespaces. Nothing to do, if pods are deleted from this
// namespace they will generate their own delete events.
func (*nodePodUpdater) OnDelete(interface{}) {}
