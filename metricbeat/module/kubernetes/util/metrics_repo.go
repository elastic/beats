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

package util

import (
	"strings"
	"sync"
)

// PodId defines a composite key for a Pod in NodeStore. A Pod is uniquely identified by a Namespace and a Name.
type PodId struct {
	Namespace string
	PodName   string
}

// Float64Metric is a wrapper for a float64 primitive type. The reason for this wrapper is to handle missing metrics with a `nil` pointer instead of a null value like `-1`. This is a better option since you could have metrics with negative values.
type Float64Metric struct {
	Value float64
}

// ContainerMetrics contains all the metrics for a Container.
type ContainerMetrics struct {
	CoresLimit  *Float64Metric
	MemoryLimit *Float64Metric
}

// NodeMetrics contains all the metrics for a Node.
type NodeMetrics struct {
	CoresAllocatable  *Float64Metric
	MemoryAllocatable *Float64Metric
}

// ContainerStore contains the name of a container and its metrics.
type ContainerStore struct {
	sync.RWMutex
	ContainerName string
	metrics       *ContainerMetrics
}

// PodStore contains the PodId of that Pod and a set of (containerName, ContainerStore) entries for each Container under a Pod.
type PodStore struct {
	sync.RWMutex
	PodId      PodId
	containers map[string]*ContainerStore
}

// NodeStore contains the name of the node, the metrics for a Node and a set of (podId, PodStore) entries for each Pod under that Node.
type NodeStore struct {
	sync.RWMutex
	NodeName string
	metrics  *NodeMetrics
	pods     map[PodId]*PodStore
}

// MetricsRepo contains a set of (nodeName, NodeStore) for each Node in the cluster.
type MetricsRepo struct {
	sync.RWMutex
	nodes map[string]*NodeStore
}

// NewPodId returns a new PodId object given a Namespace and a Pod name.
func NewPodId(namespace, podName string) PodId {
	return PodId{
		Namespace: namespace,
		PodName:   podName,
	}
}

// NewFloat64Metrics returns a Float64Metric given a float64 value.
func NewFloat64Metric(value float64) *Float64Metric {
	return &Float64Metric{
		Value: value,
	}
}

// NewContainerMetrics creates an empty ContainerMetrics object.
func NewContainerMetrics() *ContainerMetrics {
	return &ContainerMetrics{
		CoresLimit:  nil,
		MemoryLimit: nil,
	}
}

// NewNodeMetrics creates an empty NodeMetrics object.
func NewNodeMetrics() *NodeMetrics {
	return &NodeMetrics{
		CoresAllocatable:  nil,
		MemoryAllocatable: nil,
	}
}

// NewContainerStore creates an empty ContainerStore object.
func NewContainerStore(containerName string) *ContainerStore {
	ans := &ContainerStore{
		ContainerName: containerName,
		metrics:       NewContainerMetrics(),
	}
	return ans
}

// NewPodStore creates an empty PodStore object.
func NewPodStore(podId PodId) *PodStore {
	ans := &PodStore{
		PodId:      podId,
		containers: make(map[string]*ContainerStore),
	}
	return ans
}

// NewNodeStore creates an empty NodeStore object.
func NewNodeStore(nodeName string) *NodeStore {
	ans := &NodeStore{
		NodeName: nodeName,
		metrics:  NewNodeMetrics(),
		pods:     make(map[PodId]*PodStore),
	}
	return ans
}

// NewMetricsRepo creates an empty MetricsRepo object.
func NewMetricsRepo() *MetricsRepo {
	ans := &MetricsRepo{
		nodes: make(map[string]*NodeStore),
	}
	return ans
}

// Clone clones a Float64Metric object.
func (m *Float64Metric) Clone() *Float64Metric {
	return &Float64Metric{
		Value: m.Value,
	}
}

// Clone returns a copy of a ContainerMetrics object.
func (cm *ContainerMetrics) Clone() *ContainerMetrics {
	ans := NewContainerMetrics()
	if cm.CoresLimit != nil {
		ans.CoresLimit = cm.CoresLimit.Clone()
	}
	if cm.MemoryLimit != nil {
		ans.MemoryLimit = cm.MemoryLimit.Clone()
	}
	return ans
}

// Clone returns a copy of a NodeMetric object.
func (nm *NodeMetrics) Clone() *NodeMetrics {
	ans := NewNodeMetrics()
	if nm.CoresAllocatable != nil {
		ans.CoresAllocatable = nm.CoresAllocatable.Clone()
	}
	if nm.MemoryAllocatable != nil {
		ans.MemoryAllocatable = nm.MemoryAllocatable.Clone()
	}
	return ans
}

// DeleteNodeStore deletes a NodeStore from the MetricsRepo given the Node name.
func (mr *MetricsRepo) DeleteNodeStore(nodeName string) {
	mr.Lock()
	defer mr.Unlock()
	delete(mr.nodes, nodeName)
}

// DeleteAllNodeStore deletes all NodeStores from the MetricsRepo.
func (mr *MetricsRepo) DeleteAllNodeStore() {
	mr.Lock()
	defer mr.Unlock()
	for nodeName := range mr.nodes {
		delete(mr.nodes, nodeName)
	}
}

// NodeNames returns the names of all the Nodes.
func (mr *MetricsRepo) NodeNames() []string {
	mr.RLock()
	defer mr.RUnlock()
	ans := make([]string, 0, len(mr.nodes))
	for nodeName := range mr.nodes {
		ans = append(ans, nodeName)
	}
	return ans
}

// PodIds returns the names of all the Pods under a Node.
func (ns *NodeStore) PodIds() []PodId {
	ns.RLock()
	defer ns.RUnlock()
	ans := make([]PodId, 0, len(ns.pods))
	for podId := range ns.pods {
		ans = append(ans, podId)
	}
	return ans
}

// ContainerNames returns the names of all the Containers under a Pod.
func (ps *PodStore) ContainerNames() []string {
	ps.RLock()
	defer ps.RUnlock()
	ans := make([]string, 0, len(ps.containers))
	for containerName := range ps.containers {
		ans = append(ans, containerName)
	}
	return ans
}

// AddNodeStore returns/create a NodeStore given a Node name. If the NodeStore already exists, it returns the object and `false` to indicate that it didn't create a new NodeStore. Otherwise if the NodeStore doesn't exists, it creates it and it returns the new object together with `true` to indicate that it created a new NodeStore.
func (mr *MetricsRepo) AddNodeStore(nodeName string) (*NodeStore, bool) {
	mr.Lock()
	defer mr.Unlock()
	node, exists := mr.nodes[nodeName]
	if !exists {
		mr.nodes[nodeName] = NewNodeStore(nodeName)
		return mr.nodes[nodeName], true
	}
	return node, false
}

// GetNodeStore returns/create a NodeStore given a Node name. If the NodeStore already exists, it returns the object. Otherwise if the NodeStore doesn't exists, it creates an empty NodeStore and it returns it. This last behavior is to implement a [Null Object Design Pattern](https://en.wikipedia.org/wiki/Null_object_pattern).
func (mr *MetricsRepo) GetNodeStore(nodeName string) *NodeStore {
	mr.RLock()
	defer mr.RUnlock()
	ans, exists := mr.nodes[nodeName]
	if !exists {
		return NewNodeStore(nodeName)
	}
	return ans
}

// AddPodStore returns/create a PodStore given a PodId. If the PodStore already exists, it returns the object and `false` to indicate that it didn't create a new PodStore. Otherwise if the PodStore doesn't exists, it creates it and it returns the new object together with `true` to indicate that it created a new PodStore.
func (ns *NodeStore) AddPodStore(podId PodId) (*PodStore, bool) {
	ns.Lock()
	defer ns.Unlock()
	pod, exists := ns.pods[podId]
	if !exists {
		ns.pods[podId] = NewPodStore(podId)
		return ns.pods[podId], true
	}
	return pod, false
}

// GetPodStore returns/create a PodStore given a PodId. If the PodStore already exists, it returns the object. Otherwise if the PodStore doesn't exists, it creates an empty PodStore and it returns it. This last behavior is to implement a [Null Object Design Pattern](https://en.wikipedia.org/wiki/Null_object_pattern).
func (ns *NodeStore) GetPodStore(podId PodId) *PodStore {
	ns.RLock()
	defer ns.RUnlock()
	pod, exists := ns.pods[podId]
	if !exists {
		return NewPodStore(podId)
	}
	return pod
}

// DeletePodStore delete a PodStore given a PodId from a NodeStore.
func (ns *NodeStore) DeletePodStore(podId PodId) {
	ns.Lock()
	defer ns.Unlock()
	_, exists := ns.pods[podId]
	if exists {
		delete(ns.pods, podId)
	}
}

// GetNodeMetrics returns a copy of the Node metrics.
func (ns *NodeStore) GetNodeMetrics() *NodeMetrics {
	ns.RLock()
	defer ns.RUnlock()
	return ns.metrics.Clone()
}

// SetNodeMetrics set the Node metrics for a NodeStore.
func (ns *NodeStore) SetNodeMetrics(metrics *NodeMetrics) {
	ns.Lock()
	defer ns.Unlock()
	ns.metrics = metrics
}

// AddContainerStore returns/create a ContainerStore given a Container name. If the ContainerStore already exists, it returns the object and `false` to indicate that it didn't create a new ContainerStore. Otherwise if the ContainerStore doesn't exists, it creates it and it returns the new object together with `true` to indicate that it created a new ContainerStore.
func (ps *PodStore) AddContainerStore(containerName string) (*ContainerStore, bool) {
	ps.Lock()
	defer ps.Unlock()
	container, exists := ps.containers[containerName]
	if !exists {
		ps.containers[containerName] = NewContainerStore(containerName)
		return ps.containers[containerName], true
	}
	return container, false
}

// GetContainerStore returns/create a ContainerStore given a Container name. If the ContainerStore already exists, it returns the object. Otherwise if the ContainerStore doesn't exists, it creates an empty ContainerStore and it returns it. This last behavior is to implement a [Null Object Design Pattern](https://en.wikipedia.org/wiki/Null_object_pattern).
func (ps *PodStore) GetContainerStore(containerName string) *ContainerStore {
	ps.RLock()
	defer ps.RUnlock()
	container, exists := ps.containers[containerName]
	if !exists {
		return NewContainerStore(containerName)
	}
	return container
}

// SetContainerMetrics set the container metrics.
func (cs *ContainerStore) SetContainerMetrics(metrics *ContainerMetrics) {
	cs.Lock()
	defer cs.Unlock()
	cs.metrics = metrics
}

// GetContainerMetrics returns a copy of the container metrics
func (cs *ContainerStore) GetContainerMetrics() *ContainerMetrics {
	cs.RLock()
	defer cs.RUnlock()
	return cs.metrics.Clone()
}

// String concatenates Namespace and PodName by "/"
func (pi PodId) String() string {
	fields := []string{pi.Namespace, pi.PodName}
	return strings.Join(fields, "/")
}
