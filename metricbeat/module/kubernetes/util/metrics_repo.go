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

type Float64Metric struct {
	Value float64
}

type ContainerMetrics struct {
	sync.RWMutex
	CoresLimit  *Float64Metric
	MemoryLimit *Float64Metric
}

type NodeMetrics struct {
	CoresAllocatable  *Float64Metric
	MemoryAllocatable *Float64Metric
}

type PodStore struct {
	sync.RWMutex
	containers map[string]*ContainerMetrics
}

type NodeStore struct {
	sync.RWMutex
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

func NewFloat64Metric(value float64) *Float64Metric {
	return &Float64Metric{
		Value: value,
	}
}

func NewContainerMetrics() *ContainerMetrics {
	return &ContainerMetrics{
		CoresLimit:  nil,
		MemoryLimit: nil,
	}
}

func NewNodeMetrics() *NodeMetrics {
	return &NodeMetrics{
		CoresAllocatable:  nil,
		MemoryAllocatable: nil,
	}
}

func NewPodStore() *PodStore {
	ans := &PodStore{
		containers: make(map[string]*ContainerMetrics),
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

func NewMetricsRepo() *MetricsRepo {
	ans := &MetricsRepo{
		nodes: make(map[string]*NodeStore),
	}
	return ans
}

func (m *Float64Metric) Clone() *Float64Metric {
	return &Float64Metric{
		Value: m.Value,
	}
}

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

func (mr *MetricsRepo) DeleteNodeStore(nodeName string) {
	mr.Lock()
	defer mr.Unlock()
	delete(mr.nodes, nodeName)
}

func (mr *MetricsRepo) DeleteAll() {
	mr.Lock()
	defer mr.Unlock()
	for nodeName := range mr.nodes {
		delete(mr.nodes, nodeName)
	}
}

func (mr *MetricsRepo) PodNames(nodeName string) []PodId {
	mr.RLock()
	defer mr.RUnlock()
	nodeStore := mr.GetNodeStore(nodeName)
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

func (mr *MetricsRepo) AddNodeStore(nodeName string) (*NodeStore, bool) {
	mr.Lock()
	defer mr.Unlock()
	node, exists := mr.nodes[nodeName]
	if !exists {
		mr.nodes[nodeName] = NewNodeStore()
		return mr.nodes[nodeName], true
	}
	return node, false
}

func (mr *MetricsRepo) GetNodeStore(nodeName string) *NodeStore {
	mr.RLock()
	defer mr.RUnlock()
	ans, exists := mr.nodes[nodeName]
	if !exists {
		return NewNodeStore()
	}
	return ans
}

func (ns *NodeStore) AddPodStore(podId PodId) (*PodStore, bool) {
	ns.Lock()
	defer ns.Unlock()
	pod, exists := ns.pods[podId]
	if !exists {
		ns.pods[podId] = NewPodStore()
		return ns.pods[podId], true
	}
	return pod, false
}

func (ns *NodeStore) GetPodStore(podId PodId) *PodStore {
	ns.RLock()
	defer ns.RUnlock()
	pod, exists := ns.pods[podId]
	if !exists {
		return NewPodStore()
	}
	return pod
}

func (ns *NodeStore) DeletePodStore(podId PodId) {
	ns.Lock()
	defer ns.Unlock()
	_, exists := ns.pods[podId]
	if exists {
		delete(ns.pods, podId)
	}
}

func (ns *NodeStore) GetNodeMetrics() *NodeMetrics {
	ns.RLock()
	defer ns.RUnlock()
	return ns.metrics.Clone()
}

func (ns *NodeStore) SetNodeMetrics(metrics *NodeMetrics) {
	ns.Lock()
	defer ns.Unlock()
	ns.metrics = metrics
}

func (ps *PodStore) GetContainerMetrics(containerName string) *ContainerMetrics {
	ps.RLock()
	defer ps.RUnlock()
	container, exists := ps.containers[containerName]
	if !exists {
		return NewContainerMetrics()
	}
	return container.Clone()
}

func (ps *PodStore) AddContainerMetrics(containerName string) (*ContainerMetrics, bool) {
	ps.Lock()
	defer ps.Unlock()
	container, exists := ps.containers[containerName]
	if !exists {
		ps.containers[containerName] = NewContainerMetrics()
		return ps.containers[containerName], true
	}
	return container, false
}

func (cm *ContainerMetrics) SetContainerMetrics(metrics *ContainerMetrics) {
	cm.Lock()
	defer cm.Unlock()
	cm.CoresLimit = metrics.CoresLimit
	cm.MemoryLimit = metrics.MemoryLimit
}
