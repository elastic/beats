package add_kubernetes_metadata

import (
	"fmt"
	"strings"
	"sync"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

const (
	ContainerIndexerName              = "container"
	PodNameIndexerName                = "pod_name"
	IPPortIndexerName                 = "ip_port"
	EventInvolvedObjectUIDIndexerName = "event_involved_object_uid"
)

// Indexer take known resource and generate all the metadata we need to enrich
// events in a efficient way. By preindexing the metadata in the way it will be
// checked when matching events
type Indexer interface {
	// GetMetadata generates event metadata for the given resource, then returns the
	// list of indexes to create, with the metadata to put on them
	GetMetadata(r Resource) []MetadataIndex

	// GetIndexes return the list of indexes the given resource belongs to. This function
	// must return the same indexes than GetMetadata
	GetIndexes(r Resource) []string
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

//GenMeta takes in resources to generate metadata for them
type GenMeta interface {
	//GenerateMetaData generates metadata by taking in a resource as an input
	GenerateMetaData(r Resource) common.MapStr
}

func getGenMeta(indexerName string, annotations, labels, labelsExclude []string) GenMeta {
	switch indexerName {
	case EventInvolvedObjectUIDIndexerName:
		return NewGenEventMeta(annotations, labels, labelsExclude)
	}
	return NewGenDefaultMeta(annotations, labels, labelsExclude)
}

type IndexerConstructor func(config common.Config, genMeta GenMeta) (Indexer, error)

func NewIndexers(configs PluginConfig, annotations, labels, labelsExclude []string) *Indexers {
	indexers := []Indexer{}
	for _, pluginConfigs := range configs {
		for name, pluginConfig := range pluginConfigs {
			indexFunc := Indexing.GetIndexer(name)
			if indexFunc == nil {
				logp.Warn("Unable to find indexing plugin %s", name)
				continue
			}

			indexer, err := indexFunc(pluginConfig, getGenMeta(name, annotations, labels, labelsExclude))
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
func (i *Indexers) GetMetadata(r Resource) []MetadataIndex {
	var metadata []MetadataIndex
	i.RLock()
	defer i.RUnlock()
	for _, indexer := range i.indexers {
		for _, m := range indexer.GetMetadata(r) {
			metadata = append(metadata, m)
		}
	}
	return metadata
}

// GetIndexes returns the composed index list from all registered indexers
func (i *Indexers) GetIndexes(r Resource) []string {
	var indexes []string
	i.RLock()
	defer i.RUnlock()
	for _, indexer := range i.indexers {
		for _, i := range indexer.GetIndexes(r) {
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

// GenerateMetaData generates default metadata for the given resource taking to account certain filters
func (g *GenDefaultMeta) GenerateMetaData(r Resource) common.MapStr {
	labelMap := common.MapStr{}
	annotationsMap := common.MapStr{}

	if len(g.labels) == 0 {
		for k, v := range r.GetMetadata().Labels {
			labelMap[k] = v
		}
	} else {
		labelMap = generateMapSubset(r.GetMetadata().Labels, g.labels)
	}

	// Exclude any labels that are present in the exclude_labels config
	for _, label := range g.labelsExclude {
		delete(labelMap, label)
	}

	annotationsMap = generateMapSubset(r.GetMetadata().Annotations, g.annotations)

	meta := common.MapStr{
		"pod": common.MapStr{
			"name": r.GetMetadata().Name,
		},
		"namespace": r.GetMetadata().Namespace,
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

type GenEventMeta struct {
	annotations   []string
	labels        []string
	labelsExclude []string
}

func NewGenEventMeta(annotations, labels, labelsExclude []string) *GenEventMeta {
	return &GenEventMeta{
		annotations:   annotations,
		labels:        labels,
		labelsExclude: labelsExclude,
	}
}

// GenerateMetaData generates default metadata for the given resource taking to account certain filters
func (g *GenEventMeta) GenerateMetaData(r Resource) common.MapStr {
	labelMap := common.MapStr{}
	annotationsMap := common.MapStr{}

	if len(g.labels) == 0 {
		for k, v := range r.GetMetadata().Labels {
			labelMap[k] = v
		}
	} else {
		labelMap = generateMapSubset(r.GetMetadata().Labels, g.labels)
	}

	// Exclude any labels that are present in the exclude_labels config
	for _, label := range g.labelsExclude {
		delete(labelMap, label)
	}

	annotationsMap = generateMapSubset(r.GetMetadata().Annotations, g.annotations)

	involvedObject := common.MapStr{}

	if len(labelMap) != 0 {
		involvedObject["labels"] = labelMap
	}

	if len(annotationsMap) != 0 {
		involvedObject["annotations"] = annotationsMap
	}

	return common.MapStr{"event": common.MapStr{"involved_object": involvedObject}}
}

// PodNameIndexer implements default indexer based on pod name
type PodNameIndexer struct {
	genMeta GenMeta
}

func NewPodNameIndexer(_ common.Config, genMeta GenMeta) (Indexer, error) {
	return &PodNameIndexer{genMeta: genMeta}, nil
}

func (p *PodNameIndexer) GetMetadata(pod Resource) []MetadataIndex {
	data := p.genMeta.GenerateMetaData(pod)
	return []MetadataIndex{
		{
			Index: fmt.Sprintf("%s/%s", pod.GetMetadata().Namespace, pod.GetMetadata().Name),
			Data:  data,
		},
	}
}

func (p *PodNameIndexer) GetIndexes(pod Resource) []string {
	return []string{fmt.Sprintf("%s/%s", pod.GetMetadata().Namespace, pod.GetMetadata().Name)}
}

// ContainerIndexer indexes pods based on all their containers IDs
type ContainerIndexer struct {
	genMeta GenMeta
}

func NewContainerIndexer(_ common.Config, genMeta GenMeta) (Indexer, error) {
	return &ContainerIndexer{genMeta: genMeta}, nil
}

func (c *ContainerIndexer) GetMetadata(r Resource) []MetadataIndex {
	commonMeta := c.genMeta.GenerateMetaData(r)
	var metadata []MetadataIndex
	pod := r.(*Pod)
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

func (c *ContainerIndexer) GetIndexes(r Resource) []string {
	var containers []string
	pod := r.(*Pod)
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
func (h *IPPortIndexer) GetMetadata(r Resource) []MetadataIndex {
	commonMeta := h.genMeta.GenerateMetaData(r)
	pod := r.(*Pod)
	hostPorts := h.GetIndexes(pod)
	var metadata []MetadataIndex

	if pod.Status.PodIP == "" {
		return metadata
	}

	// Add pod IP
	metadata = append(metadata, MetadataIndex{
		Index: pod.Status.PodIP,
		Data:  commonMeta,
	})

	for i := 1; i < len(hostPorts); i++ {
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
func (h *IPPortIndexer) GetIndexes(r Resource) []string {
	var hostPorts []string

	pod := r.(*Pod)
	ip := pod.Status.PodIP
	if ip == "" {
		return hostPorts
	}

	// Add pod IP
	hostPorts = append(hostPorts, ip)

	for _, container := range pod.Spec.Containers {
		ports := container.Ports

		for _, port := range ports {
			if port.ContainerPort != int64(0) {
				hostPorts = append(hostPorts, fmt.Sprintf("%s:%d", ip, port.ContainerPort))
			}
		}
	}

	return hostPorts
}

// EventInvolvedObjectUIDIndexer implements default indexer based on resource uid
type EventInvolvedObjectUIDIndexer struct {
	genMeta GenMeta
}

func NewEventInvolvedObjectUIDIndexer(_ common.Config, genMeta GenMeta) (Indexer, error) {
	return &EventInvolvedObjectUIDIndexer{genMeta: genMeta}, nil
}

func (p *EventInvolvedObjectUIDIndexer) GetMetadata(r Resource) []MetadataIndex {
	return []MetadataIndex{
		{
			Index: r.GetMetadata().UID,
			Data:  p.genMeta.GenerateMetaData(r),
		},
	}
}

func (p *EventInvolvedObjectUIDIndexer) GetIndexes(r Resource) []string {
	return []string{r.GetMetadata().UID}
}
