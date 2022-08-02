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
	"sync"
)

type PodId struct {
	Namespace string
	PodName   string
}
type ContainerId struct {
	PodId
	ContainerName string
}

type ContainerMetrics struct {
	CoresLimit  float64
	MemoryLimit float64
}

type NodeMetrics struct {
	CoresAllocatable  float64
	MemoryAllocatable float64
}

type PodStore struct {
	// sync.RWMutex
	containers map[string]*ContainerMetrics
}

type NodeStore struct {
	// sync.RWMutex
	metrics *NodeMetrics
	pods    map[PodId]*PodStore
}

type MetricsRepo struct {
	sync.RWMutex
	nodes map[string]*NodeStore
}

func NewPodId(namespace, podName string) PodId {
	return PodId{
		Namespace: namespace,
		PodName:   podName,
	}
}

func NewContainerId(podId PodId, containerName string) ContainerId {
	return ContainerId{
		PodId:         podId,
		ContainerName: containerName,
	}
}

func NewContainerMetrics() *ContainerMetrics {
	return &ContainerMetrics{
		CoresLimit:  -1,
		MemoryLimit: -1,
	}
}

func (cm *ContainerMetrics) set(metrics *ContainerMetrics) {
	cm.CoresLimit = metrics.CoresLimit
	cm.MemoryLimit = metrics.MemoryLimit
}

func (cm *ContainerMetrics) clone() *ContainerMetrics {
	ans := &ContainerMetrics{
		CoresLimit:  cm.CoresLimit,
		MemoryLimit: cm.MemoryLimit,
	}
	return ans
}

func NewNodeMetrics() *NodeMetrics {
	return &NodeMetrics{
		CoresAllocatable:  -1,
		MemoryAllocatable: -1,
	}
}

func (nm *NodeMetrics) set(metrics *NodeMetrics) {
	nm.CoresAllocatable = metrics.CoresAllocatable
	nm.MemoryAllocatable = metrics.MemoryAllocatable
}

func (nm *NodeMetrics) clone() *NodeMetrics {
	ans := &NodeMetrics{
		CoresAllocatable:  nm.CoresAllocatable,
		MemoryAllocatable: nm.MemoryAllocatable,
	}
	return ans
}

func NewMetricsRepo() *MetricsRepo {
	ans := &MetricsRepo{
		nodes: make(map[string]*NodeStore),
	}
	return ans
}

func NewNodeStore() *NodeStore {
	ans := &NodeStore{
		metrics: NewNodeMetrics(),
		pods:    make(map[PodId]*PodStore),
	}
	return ans
}

func NewPodStore() *PodStore {
	ans := &PodStore{
		containers: make(map[string]*ContainerMetrics),
	}
	return ans
}

func (mr *MetricsRepo) SetNodeMetrics(nodeName string, metrics *NodeMetrics) {
	mr.Lock()
	defer mr.Unlock()
	nodeStore, _ := mr.add(nodeName)
	nodeStore.metrics.set(metrics)
}

func (mr *MetricsRepo) SetContainerMetrics(nodeName string, containerId ContainerId, metrics *ContainerMetrics) {
	mr.Lock()
	defer mr.Unlock()
	nodeStore, _ := mr.add(nodeName)
	podStore, _ := nodeStore.add(containerId.PodId)
	podStore.setContainerMetrics(containerId.ContainerName, metrics)
}

func (mr *MetricsRepo) GetNodeMetrics(nodeName string) *NodeMetrics {
	mr.RLock()
	defer mr.RUnlock()
	nodeStore := mr.get(nodeName)
	if nodeStore == nil {
		return NewNodeMetrics()
	}
	return nodeStore.metrics.clone()
}

func (mr *MetricsRepo) GetContainerMetrics(nodeName string, containerId ContainerId) *ContainerMetrics {
	mr.RLock()
	defer mr.RUnlock()
	nodeStore := mr.get(nodeName)
	if nodeStore == nil {
		return NewContainerMetrics()
	}
	podStore := nodeStore.get(containerId.PodId)
	if podStore == nil {
		return NewContainerMetrics()
	}
	return podStore.get(containerId.ContainerName)
}

func (mr *MetricsRepo) add(nodeName string) (*NodeStore, bool) {
	node, exists := mr.nodes[nodeName]
	if !exists {
		mr.nodes[nodeName] = NewNodeStore()
		return mr.nodes[nodeName], true
	}
	return node, false
}

func (mr *MetricsRepo) get(nodeName string) *NodeStore {
	ans, exists := mr.nodes[nodeName]
	if !exists {
		return nil
	}
	return ans
}

func (ns *NodeStore) add(podId PodId) (*PodStore, bool) {
	pod, exists := ns.pods[podId]
	if !exists {
		ns.pods[podId] = NewPodStore()
		return ns.pods[podId], true
	}
	return pod, false
}

func (ns *NodeStore) get(podId PodId) *PodStore {
	pod, exists := ns.pods[podId]
	if !exists {
		return nil
	}
	return pod
}

func (ps *PodStore) get(containerName string) *ContainerMetrics {
	container, exists := ps.containers[containerName]
	if !exists {
		return nil
	}
	return container.clone()
}

func (ps *PodStore) setContainerMetrics(containerName string, metrics *ContainerMetrics) {
	container, exists := ps.containers[containerName]
	if !exists {
		ps.containers[containerName] = NewContainerMetrics()
		container = ps.containers[containerName]
	}
	container.set(metrics)
}

func (mr *MetricsRepo) DeleteAllNodes() {
	mr.Lock()
	defer mr.Unlock()
	for nodeName := range mr.nodes {
		delete(mr.nodes, nodeName)
	}
}

func (mr *MetricsRepo) PodNames(nodeName string) []PodId {
	mr.RLock()
	defer mr.RUnlock()
	nodeStore := mr.get(nodeName)
	if nodeStore == nil {
		return []PodId{}
	}

	ans := make([]PodId, 0, len(nodeStore.pods))
	for podId := range nodeStore.pods {
		ans = append(ans, podId)
	}
	return ans
}

func (mr *MetricsRepo) NodeNames() []string {
	mr.RLock()
	defer mr.RUnlock()
	ans := make([]string, 0, len(mr.nodes))
	for nodeName := range mr.nodes {
		ans = append(ans, nodeName)
	}
	return ans
}

func (mr *MetricsRepo) DeleteNode(nodeName string) {
	mr.Lock()
	defer mr.Unlock()
	delete(mr.nodes, nodeName)
}

func (mr *MetricsRepo) DeletePod(nodeName string, podId PodId) {
	mr.Lock()
	defer mr.Unlock()
	node, exists := mr.nodes[nodeName]
	if exists {
		delete(node.pods, podId)
	}
}
