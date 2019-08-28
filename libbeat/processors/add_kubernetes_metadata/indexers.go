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

package add_kubernetes_metadata

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/kubernetes"
	"github.com/elastic/beats/libbeat/logp"
)

const (
	ContainerIndexerName = "container"
	PodNameIndexerName   = "pod_name"
	PodUIDIndexerName    = "pod_uid"
	IPPortIndexerName    = "ip_port"
)

// Indexer take known pods and generate all the metadata we need to enrich
// events in a efficient way. By preindexing the metadata in the way it will be
// checked when matching events
type Indexer interface {
	// GetMetadata generates event metadata for the given pod, then returns the
	// list of indexes to create, with the metadata to put on them
	GetMetadata(pod *kubernetes.Pod) []MetadataIndex

	// GetIndexes return the list of indexes the given pod belongs to. This function
	// must return the same indexes than GetMetadata
	GetIndexes(pod *kubernetes.Pod) []string
}

// MetadataIndex holds a pair of index -> metadata info
type MetadataIndex struct {
	Index string
	Data  common.MapStr
}

type Indexers struct {
	sync.RWMutex
	indexers []Indexer
}

// IndexerConstructor builds a new indexer from its settings
type IndexerConstructor func(config common.Config, metaGen kubernetes.MetaGenerator) (Indexer, error)

// NewIndexers builds indexers object
func NewIndexers(configs PluginConfig, metaGen kubernetes.MetaGenerator) *Indexers {
	indexers := []Indexer{}
	for _, pluginConfigs := range configs {
		for name, pluginConfig := range pluginConfigs {
			indexFunc := Indexing.GetIndexer(name)
			if indexFunc == nil {
				logp.Warn("Unable to find indexing plugin %s", name)
				continue
			}

			indexer, err := indexFunc(pluginConfig, metaGen)
			if err != nil {
				logp.Warn("Unable to initialize indexing plugin %s due to error %v", name, err)
				continue
			}

			indexers = append(indexers, indexer)
		}
	}

	return &Indexers{
		indexers: indexers,
	}
}

// GetIndexes returns the composed index list from all registered indexers
func (i *Indexers) GetIndexes(pod *kubernetes.Pod) []string {
	var indexes []string
	i.RLock()
	defer i.RUnlock()
	for _, indexer := range i.indexers {
		for _, i := range indexer.GetIndexes(pod) {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

// GetMetadata returns the composed metadata list from all registered indexers
func (i *Indexers) GetMetadata(pod *kubernetes.Pod) []MetadataIndex {
	var metadata []MetadataIndex
	i.RLock()
	defer i.RUnlock()
	for _, indexer := range i.indexers {
		for _, m := range indexer.GetMetadata(pod) {
			metadata = append(metadata, m)
		}
	}
	return metadata
}

// Empty returns true if indexers list is empty
func (i *Indexers) Empty() bool {
	i.RLock()
	defer i.RUnlock()
	if len(i.indexers) == 0 {
		return true
	}

	return false
}

// PodNameIndexer implements default indexer based on pod name
type PodNameIndexer struct {
	metaGen kubernetes.MetaGenerator
}

// NewPodNameIndexer initializes and returns a PodNameIndexer
func NewPodNameIndexer(_ common.Config, metaGen kubernetes.MetaGenerator) (Indexer, error) {
	return &PodNameIndexer{metaGen: metaGen}, nil
}

// GetMetadata returns metadata for the given pod, if it matches the index
func (p *PodNameIndexer) GetMetadata(pod *kubernetes.Pod) []MetadataIndex {
	data := p.metaGen.PodMetadata(pod)
	return []MetadataIndex{
		{
			Index: fmt.Sprintf("%s/%s", pod.GetObjectMeta().GetNamespace(), pod.GetObjectMeta().GetName()),
			Data:  data,
		},
	}
}

// GetIndexes returns the indexes for the given Pod
func (p *PodNameIndexer) GetIndexes(pod *kubernetes.Pod) []string {
	return []string{fmt.Sprintf("%s/%s", pod.GetObjectMeta().GetNamespace(), pod.GetObjectMeta().GetName())}
}

// PodUIDIndexer indexes pods based on the pod UID
type PodUIDIndexer struct {
	metaGen kubernetes.MetaGenerator
}

// NewPodUIDIndexer initializes and returns a PodUIDIndexer
func NewPodUIDIndexer(_ common.Config, metaGen kubernetes.MetaGenerator) (Indexer, error) {
	return &PodUIDIndexer{metaGen: metaGen}, nil
}

// GetMetadata returns the composed metadata from PodNameIndexer and the pod UID
func (p *PodUIDIndexer) GetMetadata(pod *kubernetes.Pod) []MetadataIndex {
	data := p.metaGen.PodMetadata(pod)
	return []MetadataIndex{
		{
			Index: string(pod.GetObjectMeta().GetUID()),
			Data:  data,
		},
	}
}

// GetIndexes returns the indexes for the given Pod
func (p *PodUIDIndexer) GetIndexes(pod *kubernetes.Pod) []string {
	return []string{string(pod.GetObjectMeta().GetUID())}
}

// ContainerIndexer indexes pods based on all their containers IDs
type ContainerIndexer struct {
	metaGen kubernetes.MetaGenerator
}

// NewContainerIndexer initializes and returns a ContainerIndexer
func NewContainerIndexer(_ common.Config, metaGen kubernetes.MetaGenerator) (Indexer, error) {
	return &ContainerIndexer{metaGen: metaGen}, nil
}

// GetMetadata returns the composed metadata list from all registered indexers
func (c *ContainerIndexer) GetMetadata(pod *kubernetes.Pod) []MetadataIndex {
	var metadata []MetadataIndex
	for _, status := range append(pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses...) {
		cID := kubernetes.ContainerID(status)
		if cID == "" {
			continue
		}
		metadata = append(metadata, MetadataIndex{
			Index: cID,
			Data:  c.metaGen.ContainerMetadata(pod, status.Name, status.Image),
		})
	}

	return metadata
}

// GetIndexes returns the indexes for the given Pod
func (c *ContainerIndexer) GetIndexes(pod *kubernetes.Pod) []string {
	var containers []string
	for _, status := range append(pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses...) {
		cID := kubernetes.ContainerID(status)
		if cID == "" {
			continue
		}
		containers = append(containers, cID)
	}
	return containers
}

// IPPortIndexer indexes pods based on all their host:port combinations
type IPPortIndexer struct {
	metaGen kubernetes.MetaGenerator
}

// NewIPPortIndexer creates and returns a new indexer for pod IP & ports
func NewIPPortIndexer(_ common.Config, metaGen kubernetes.MetaGenerator) (Indexer, error) {
	return &IPPortIndexer{metaGen: metaGen}, nil
}

// GetMetadata returns metadata for the given pod, if it matches the index
func (h *IPPortIndexer) GetMetadata(pod *kubernetes.Pod) []MetadataIndex {
	var metadata []MetadataIndex

	if pod.Status.PodIP == "" {
		return metadata
	}

	// Add pod IP
	metadata = append(metadata, MetadataIndex{
		Index: pod.Status.PodIP,
		Data:  h.metaGen.PodMetadata(pod),
	})

	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			if port.ContainerPort != 0 {

				metadata = append(metadata, MetadataIndex{
					Index: fmt.Sprintf("%s:%d", pod.Status.PodIP, port.ContainerPort),
					Data:  h.metaGen.ContainerMetadata(pod, container.Name, container.Image),
				})
			}
		}
	}

	return metadata
}

// GetIndexes returns the indexes for the given Pod
func (h *IPPortIndexer) GetIndexes(pod *kubernetes.Pod) []string {
	var hostPorts []string

	if pod.Status.PodIP == "" {
		return hostPorts
	}

	// Add pod IP
	hostPorts = append(hostPorts, pod.Status.PodIP)

	for _, container := range pod.Spec.Containers {
		ports := container.Ports

		for _, port := range ports {
			if port.ContainerPort != 0 {
				hostPorts = append(hostPorts, fmt.Sprintf("%s:%d", pod.Status.PodIP, port.ContainerPort))
			}
		}
	}

	return hostPorts
}
