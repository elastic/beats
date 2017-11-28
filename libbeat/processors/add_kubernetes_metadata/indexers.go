package add_kubernetes_metadata

import (
	"fmt"
	"strings"
	"sync"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

const (
	ContainerIndexerName = "container"
	PodNameIndexerName   = "pod_name"
	IPPortIndexerName    = "ip_port"
)

// Indexer take known pods and generate all the metadata we need to enrich
// events in a efficient way. By preindexing the metadata in the way it will be
// checked when matching events
type Indexer interface {
	// GetMetadata generates event metadata for the given pod, then returns the
	// list of indexes to create, with the metadata to put on them
	GetMetadata(pod *Pod) []MetadataIndex

	// GetIndexes return the list of indexes the given pod belongs to. This function
	// must return the same indexes than GetMetadata
	GetIndexes(pod *Pod) []string
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

//GenMeta takes in pods to generate metadata for them
type GenMeta interface {
	//GenerateMetaData generates metadata by taking in a pod as an input
	GenerateMetaData(pod *Pod) common.MapStr
}

type IndexerConstructor func(config common.Config, genMeta GenMeta) (Indexer, error)

func NewIndexers(configs PluginConfig, metaGen *GenDefaultMeta) *Indexers {
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
			}

			indexers = append(indexers, indexer)
		}
	}

	return &Indexers{
		indexers: indexers,
	}
}

// GetMetadata returns the composed metadata list from all registered indexers
func (i *Indexers) GetMetadata(pod *Pod) []MetadataIndex {
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

// GetIndexes returns the composed index list from all registered indexers
func (i *Indexers) GetIndexes(pod *Pod) []string {
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

func (i *Indexers) Empty() bool {
	if len(i.indexers) == 0 {
		return true
	}

	return false
}

type GenDefaultMeta struct {
	annotations   []string
	labels        []string
	labelsExclude []string
}

func NewGenDefaultMeta(annotations, labels, labelsExclude []string) *GenDefaultMeta {
	return &GenDefaultMeta{
		annotations:   annotations,
		labels:        labels,
		labelsExclude: labelsExclude,
	}
}

// GenerateMetaData generates default metadata for the given pod taking to account certain filters
func (g *GenDefaultMeta) GenerateMetaData(pod *Pod) common.MapStr {
	labelMap := common.MapStr{}
	annotationsMap := common.MapStr{}

	if len(g.labels) == 0 {
		for k, v := range pod.Metadata.Labels {
			labelMap[k] = v
		}
	} else {
		labelMap = generateMapSubset(pod.Metadata.Labels, g.labels)
	}

	// Exclude any labels that are present in the exclude_labels config
	for _, label := range g.labelsExclude {
		delete(labelMap, label)
	}

	annotationsMap = generateMapSubset(pod.Metadata.Annotations, g.annotations)

	meta := common.MapStr{
		"pod": common.MapStr{
			"name": pod.Metadata.Name,
		},
		"namespace": pod.Metadata.Namespace,
	}

	if len(labelMap) != 0 {
		meta["labels"] = labelMap
	}

	if len(annotationsMap) != 0 {
		meta["annotations"] = annotationsMap
	}

	return meta
}

func generateMapSubset(input map[string]string, keys []string) common.MapStr {
	output := common.MapStr{}
	if input == nil {
		return output
	}

	for _, key := range keys {
		value, ok := input[key]
		if ok {
			output[key] = value
		}
	}

	return output
}

// PodNameIndexer implements default indexer based on pod name
type PodNameIndexer struct {
	genMeta GenMeta
}

func NewPodNameIndexer(_ common.Config, genMeta GenMeta) (Indexer, error) {
	return &PodNameIndexer{genMeta: genMeta}, nil
}

func (p *PodNameIndexer) GetMetadata(pod *Pod) []MetadataIndex {
	data := p.genMeta.GenerateMetaData(pod)
	return []MetadataIndex{
		{
			Index: fmt.Sprintf("%s/%s", pod.Metadata.Namespace, pod.Metadata.Name),
			Data:  data,
		},
	}
}

func (p *PodNameIndexer) GetIndexes(pod *Pod) []string {
	return []string{fmt.Sprintf("%s/%s", pod.Metadata.Namespace, pod.Metadata.Name)}
}

// ContainerIndexer indexes pods based on all their containers IDs
type ContainerIndexer struct {
	genMeta GenMeta
}

func NewContainerIndexer(_ common.Config, genMeta GenMeta) (Indexer, error) {
	return &ContainerIndexer{genMeta: genMeta}, nil
}

func (c *ContainerIndexer) GetMetadata(pod *Pod) []MetadataIndex {
	commonMeta := c.genMeta.GenerateMetaData(pod)
	var metadata []MetadataIndex
	for _, status := range append(pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses...) {
		cID := containerID(status)
		if cID == "" {
			continue
		}

		containerMeta := commonMeta.Clone()
		containerMeta["container"] = common.MapStr{
			"name": status.Name,
		}
		metadata = append(metadata, MetadataIndex{
			Index: cID,
			Data:  containerMeta,
		})
	}

	return metadata
}

func (c *ContainerIndexer) GetIndexes(pod *Pod) []string {
	var containers []string
	for _, status := range append(pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses...) {
		cID := containerID(status)
		if cID == "" {
			continue
		}
		containers = append(containers, cID)
	}
	return containers
}

func containerID(status PodContainerStatus) string {
	cID := status.ContainerID
	if cID != "" {
		parts := strings.Split(cID, "//")
		if len(parts) == 2 {
			return parts[1]
		}
	}
	return ""
}

// IPPortIndexer indexes pods based on all their host:port combinations
type IPPortIndexer struct {
	genMeta GenMeta
}

// NewIPPortIndexer creates and returns a new indexer for pod IP & ports
func NewIPPortIndexer(_ common.Config, genMeta GenMeta) (Indexer, error) {
	return &IPPortIndexer{genMeta: genMeta}, nil
}

// GetMetadata returns metadata for the given pod, if it matches the index
func (h *IPPortIndexer) GetMetadata(pod *Pod) []MetadataIndex {
	commonMeta := h.genMeta.GenerateMetaData(pod)
	hostPorts := h.GetIndexes(pod)
	var metadata []MetadataIndex

	if pod.Status.PodIP == "" {
		return metadata
	}
	for i := 0; i < len(hostPorts); i++ {
		dobreak := false
		containerMeta := commonMeta.Clone()
		for _, container := range pod.Spec.Containers {
			ports := container.Ports

			for _, port := range ports {
				if port.ContainerPort == int64(0) {
					continue
				}
				if strings.Index(hostPorts[i], fmt.Sprintf("%s:%d", pod.Status.PodIP, port.ContainerPort)) != -1 {
					containerMeta["container"] = common.MapStr{
						"name": container.Name,
					}
					dobreak = true
					break
				}
			}

			if dobreak {
				break
			}

		}

		metadata = append(metadata, MetadataIndex{
			Index: hostPorts[i],
			Data:  containerMeta,
		})
	}

	return metadata
}

// GetIndexes returns the indexes for the given Pod
func (h *IPPortIndexer) GetIndexes(pod *Pod) []string {
	var hostPorts []string

	ip := pod.Status.PodIP
	if ip == "" {
		return hostPorts
	}
	for _, container := range pod.Spec.Containers {
		ports := container.Ports

		for _, port := range ports {
			if port.ContainerPort != int64(0) {
				hostPorts = append(hostPorts, fmt.Sprintf("%s:%d", ip, port.ContainerPort))
			} else {
				hostPorts = append(hostPorts, ip)
			}

		}

	}

	return hostPorts
}
