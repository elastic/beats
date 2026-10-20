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
	"net"
	"strconv"
	"strings"

	"github.com/elastic/beats/v7/pkg/autodiscover/kubernetes"
	"github.com/elastic/beats/v7/pkg/autodiscover/kubernetes/metadata"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	ContainerIndexerName = "container"
	PodNameIndexerName   = "pod_name"
	PodUIDIndexerName    = "pod_uid"
	IPPortIndexerName    = "ip_port"
)

const (
	// ImageMetadataSourceKey is the setting that selects where
	// kubernetes.container.image comes from. It is set on the processor and
	// forwarded to the indexers that report container images.
	ImageMetadataSourceKey = "image_metadata_source"

	// ImageSourceStatus uses the image from the container status, as resolved by the runtime.
	ImageSourceStatus = "status"
	// ImageSourceSpec uses the image from the pod spec, as written in the manifest.
	ImageSourceSpec = "spec"
)

// imageSourceConfig is the part of an indexer config that selects the image source.
type imageSourceConfig struct {
	Source string `config:"image_metadata_source"`
}

func (c *imageSourceConfig) Validate() error {
	switch c.Source {
	case "", ImageSourceStatus, ImageSourceSpec:
		return nil
	}
	return fmt.Errorf("invalid %s %q, valid values include `%s`, `%s`", ImageMetadataSourceKey, c.Source, ImageSourceStatus, ImageSourceSpec)
}

// useSpecImage reports whether the indexer config selects the pod spec image.
func useSpecImage(cfg config.C) (bool, error) {
	var c imageSourceConfig
	if err := cfg.Unpack(&c); err != nil {
		return false, err
	}
	return c.Source == ImageSourceSpec, nil
}

// Indexer take known pods and generate all the metadata we need to enrich
// events in a efficient way. By preindexing the metadata in the way it will be
// checked when matching events
type Indexer interface {
	// GetMetadata generates event metadata for the given pod, then returns the
	// list of indexes to create, with the metadata to put on them.
	GetMetadata(pod *kubernetes.Pod) []MetadataIndex

	// GetIndexes returns the list of indexes the given pod belongs to.
	// Must return the same set as the Index fields in GetMetadata.
	// Not called by the annotator in production; exists for tests and out-of-tree implementors.
	GetIndexes(pod *kubernetes.Pod) []string
}

// MetadataIndex holds a pair of index -> metadata info
type MetadataIndex struct {
	Index string
	Data  mapstr.M
}

type Indexers struct {
	indexers []Indexer
}

// IndexerConstructor builds a new indexer from its settings
type IndexerConstructor func(config config.C, metaGen metadata.MetaGen) (Indexer, error)

// NewIndexers builds indexers object
// imageSource, when not empty, is forwarded to every indexer config that does
// not set its own image_metadata_source.
func NewIndexers(configs PluginConfig, metaGen metadata.MetaGen, imageSource string) *Indexers {
	indexers := []Indexer{}
	for _, pluginConfigs := range configs {
		for name, pluginConfig := range pluginConfigs {
			indexFunc := Indexing.GetIndexer(name)
			if indexFunc == nil {
				logp.Warn("Unable to find indexing plugin %s", name)
				continue
			}

			if imageSource != "" && !pluginConfig.HasField(ImageMetadataSourceKey) {
				if err := pluginConfig.SetString(ImageMetadataSourceKey, -1, imageSource); err != nil {
					logp.Warn("Unable to set %s for indexing plugin %s due to error %v", ImageMetadataSourceKey, name, err)
				}
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

// GetMetadata returns the composed metadata list from all registered indexers
func (i *Indexers) GetMetadata(pod *kubernetes.Pod) []MetadataIndex {
	var metadata []MetadataIndex
	for _, indexer := range i.indexers {
		metadata = append(metadata, indexer.GetMetadata(pod)...)
	}
	return metadata
}

// Empty returns true if indexers list is empty
func (i *Indexers) Empty() bool {
	return len(i.indexers) == 0
}

// PodNameIndexer implements default indexer based on pod name
type PodNameIndexer struct {
	metaGen metadata.MetaGen
}

// NewPodNameIndexer initializes and returns a PodNameIndexer
func NewPodNameIndexer(_ config.C, metaGen metadata.MetaGen) (Indexer, error) {
	return &PodNameIndexer{metaGen: metaGen}, nil
}

// GetMetadata returns metadata for the given pod, if it matches the index
func (p *PodNameIndexer) GetMetadata(pod *kubernetes.Pod) []MetadataIndex {
	data := p.metaGen.Generate(pod)
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
	metaGen   metadata.MetaGen
	specImage bool
}

// NewPodUIDIndexer initializes and returns a PodUIDIndexer
func NewPodUIDIndexer(cfg config.C, metaGen metadata.MetaGen) (Indexer, error) {
	specImage, err := useSpecImage(cfg)
	if err != nil {
		return nil, err
	}
	return &PodUIDIndexer{metaGen: metaGen, specImage: specImage}, nil
}

// GetMetadata returns entries for uid, uid/container, and uid/container/restartCount
// so the logs_path matcher (resource_type:pod) can resolve container-level fields.
func (p *PodUIDIndexer) GetMetadata(pod *kubernetes.Pod) []MetadataIndex {
	uid := string(pod.GetObjectMeta().GetUID())

	baseMeta := p.metaGen.Generate(pod)
	m := []MetadataIndex{{Index: uid, Data: baseMeta}}

	for _, c := range kubernetes.GetContainersInPod(pod) {
		// status image includes the digest; spec is the fallback
		image := c.Spec.Image
		if !p.specImage && c.Status.Image != "" {
			image = c.Status.Image
		}
		containerIdx := uid + "/" + c.Spec.Name

		// <uid>/<container>: name + image, valid across all restarts
		containerMeta := baseMeta.Clone()
		_, _ = containerMeta.Put("kubernetes.container.name", c.Spec.Name)
		_, _ = containerMeta.Put("kubernetes.container.image", image)
		m = append(m, MetadataIndex{Index: containerIdx, Data: containerMeta})

		// <uid>/<container>/<restartCount>: also includes container id and runtime
		if c.ID != "" {
			idMeta := containerMeta.Clone()
			_, _ = idMeta.Put("kubernetes.container.id", c.ID)
			_, _ = idMeta.Put("kubernetes.container.runtime", c.Runtime)
			m = append(m, MetadataIndex{
				Index: fmt.Sprintf("%s/%d", containerIdx, c.Status.RestartCount),
				Data:  idMeta,
			})
		}

		// <uid>/<container>/<restartCount-1>: previous container id/runtime for rotated log files
		if c.Status.RestartCount > 0 {
			if term := c.Status.LastTerminationState.Terminated; term != nil && term.ContainerID != "" {
				parts := strings.SplitN(term.ContainerID, "://", 2)
				if len(parts) == 2 {
					prevID, prevRuntime := parts[1], parts[0]
					// container.image here is the current image; k8s doesn't expose the previous image
					prevMeta := containerMeta.Clone()
					_, _ = prevMeta.Put("kubernetes.container.id", prevID)
					_, _ = prevMeta.Put("kubernetes.container.runtime", prevRuntime)
					m = append(m, MetadataIndex{
						Index: fmt.Sprintf("%s/%d", containerIdx, c.Status.RestartCount-1),
						Data:  prevMeta,
					})
				}
			}
		}
	}

	return m
}

// GetIndexes returns the indexes for the given Pod.
// Derived from GetMetadata to guarantee the two sets stay in sync.
func (p *PodUIDIndexer) GetIndexes(pod *kubernetes.Pod) []string {
	m := p.GetMetadata(pod)
	indexes := make([]string, 0, len(m))
	for _, mi := range m {
		indexes = append(indexes, mi.Index)
	}
	return indexes
}

// ContainerIndexer indexes pods based on all their containers IDs
type ContainerIndexer struct {
	metaGen   metadata.MetaGen
	specImage bool
}

// NewContainerIndexer initializes and returns a ContainerIndexer
func NewContainerIndexer(cfg config.C, metaGen metadata.MetaGen) (Indexer, error) {
	specImage, err := useSpecImage(cfg)
	if err != nil {
		return nil, err
	}
	return &ContainerIndexer{metaGen: metaGen, specImage: specImage}, nil
}

// GetMetadata returns the composed metadata list from all registered indexers
func (c *ContainerIndexer) GetMetadata(pod *kubernetes.Pod) []MetadataIndex {
	var m []MetadataIndex
	var specImages map[string]string
	if c.specImage {
		specImages = specImagesInPod(pod)
	}
	for _, status := range getContainerStatusesInPod(pod) {
		cID, runtime := kubernetes.ContainerIDWithRuntime(status)
		if cID == "" {
			continue
		}
		image := status.Image
		if specImage, ok := specImages[status.Name]; ok {
			image = specImage
		}
		m = append(m, MetadataIndex{
			Index: cID,
			Data: c.metaGen.Generate(
				pod,
				metadata.WithFields("container.name", status.Name),
				metadata.WithFields("container.image", image),
				metadata.WithFields("container.id", cID),
				metadata.WithFields("container.runtime", runtime),
			),
		})
	}

	return m
}

// GetIndexes returns the indexes for the given Pod
func (c *ContainerIndexer) GetIndexes(pod *kubernetes.Pod) []string {
	var containers []string
	for _, status := range getContainerStatusesInPod(pod) {
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
	metaGen metadata.MetaGen
}

// NewIPPortIndexer creates and returns a new indexer for pod IP & ports
func NewIPPortIndexer(_ config.C, metaGen metadata.MetaGen) (Indexer, error) {
	return &IPPortIndexer{metaGen: metaGen}, nil
}

// GetMetadata returns metadata for the given pod, if it matches the index
func (h *IPPortIndexer) GetMetadata(pod *kubernetes.Pod) []MetadataIndex {
	var m []MetadataIndex

	if pod.Status.PodIP == "" {
		return m
	}

	// Add pod IP
	m = append(m, MetadataIndex{
		Index: pod.Status.PodIP,
		Data:  h.metaGen.Generate(pod),
	})

	cIDs := make(map[string]string)
	runtimes := make(map[string]string)
	for _, status := range getContainerStatusesInPod(pod) {
		cID, runtime := kubernetes.ContainerIDWithRuntime(status)
		if cID == "" {
			continue
		}
		cIDs[status.Name] = cID
		runtimes[status.Name] = runtime
	}

	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			if port.ContainerPort != 0 {

				m = append(m, MetadataIndex{
					Index: net.JoinHostPort(pod.Status.PodIP, strconv.Itoa(int(port.ContainerPort))),
					Data: h.metaGen.Generate(
						pod,
						metadata.WithFields("container.name", container.Name),
						metadata.WithFields("container.image", container.Image),
						metadata.WithFields("container.id", cIDs[container.Name]),
						metadata.WithFields("container.runtime", runtimes[container.Name]),
					),
				})
			}
		}
	}

	return m
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
				hostPorts = append(hostPorts, net.JoinHostPort(pod.Status.PodIP, strconv.Itoa(int(port.ContainerPort))))
			}
		}
	}

	return hostPorts
}

// specImagesInPod maps container names to the image in the pod spec.
func specImagesInPod(pod *kubernetes.Pod) map[string]string {
	images := make(map[string]string)
	if pod == nil {
		return images
	}
	for _, c := range pod.Spec.Containers {
		images[c.Name] = c.Image
	}
	for _, c := range pod.Spec.InitContainers {
		images[c.Name] = c.Image
	}
	for _, c := range pod.Spec.EphemeralContainers {
		images[c.Name] = c.Image
	}
	return images
}

func getContainerStatusesInPod(pod *kubernetes.Pod) []kubernetes.PodContainerStatus {
	if pod == nil {
		return nil
	}
	var statuses []kubernetes.PodContainerStatus
	statuses = append(statuses, pod.Status.ContainerStatuses...)
	statuses = append(statuses, pod.Status.InitContainerStatuses...)
	statuses = append(statuses, pod.Status.EphemeralContainerStatuses...)
	return statuses
}
