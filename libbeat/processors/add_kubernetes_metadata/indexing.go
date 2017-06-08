package add_kubernetes_metadata

import (
	"fmt"
	"strings"
	"sync"

	"github.com/elastic/beats/libbeat/common"
)

//Names of indexers and matchers that have been defined.
const (
	PodNameIndexerName   = "pod_name"
	FieldMatcherName     = "fields"
	ContainerIndexerName = "container"
)

// Indexing is the singleton Register instance where all Indexers and Matchers
// are stored
var Indexing = NewRegister()

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

// Matcher takes a new event and returns the index
type Matcher interface {
	// MetadataIndex returns the index string to use in annotation lookups for the given
	// event. A previous indexer should have generated that index for this to work
	// This function can return "" if the event doesn't match
	MetadataIndex(event common.MapStr) string
}

//GenMeta takes in pods to generate metadata for them
type GenMeta interface {
	//GenerateMetaData generates metadata by taking in a pod as an input
	GenerateMetaData(pod *Pod) common.MapStr
}

type Indexers struct {
	sync.RWMutex
	indexers []Indexer
}

type Matchers struct {
	sync.RWMutex
	matchers []Matcher
}

// Register contains Indexer and Matchers to use on pod indexing and event matching
type Register struct {
	sync.RWMutex
	indexers map[string]IndexerConstructor
	matchers map[string]MatcherConstructor

	defaultIndexerConfigs map[string]common.Config
	defaultMatcherConfigs map[string]common.Config
}

type IndexerConstructor func(config common.Config, genMeta GenMeta) (Indexer, error)
type MatcherConstructor func(config common.Config) (Matcher, error)

// NewRegister creates and returns a new Register.
func NewRegister() *Register {
	return &Register{
		indexers: make(map[string]IndexerConstructor, 0),
		matchers: make(map[string]MatcherConstructor, 0),

		defaultIndexerConfigs: make(map[string]common.Config, 0),
		defaultMatcherConfigs: make(map[string]common.Config, 0),
	}
}

// AddIndexer to the register
func (r *Register) AddIndexer(name string, indexer IndexerConstructor) {
	r.RWMutex.Lock()
	defer r.RWMutex.Unlock()
	r.indexers[name] = indexer
}

// AddMatcher to the register
func (r *Register) AddMatcher(name string, matcher MatcherConstructor) {
	r.RWMutex.Lock()
	defer r.RWMutex.Unlock()
	r.matchers[name] = matcher
}

// AddIndexer to the register
func (r *Register) AddDefaultIndexerConfig(name string, config common.Config) {
	r.defaultIndexerConfigs[name] = config
}

// AddMatcher to the register
func (r *Register) AddDefaultMatcherConfig(name string, config common.Config) {
	r.defaultMatcherConfigs[name] = config
}

// AddIndexer to the register
func (r *Register) GetIndexer(name string) IndexerConstructor {
	indexer, ok := r.indexers[name]
	if ok {
		return indexer
	} else {
		return nil
	}
}

// AddMatcher to the register
func (r *Register) GetMatcher(name string) MatcherConstructor {
	matcher, ok := r.matchers[name]
	if ok {
		return matcher
	} else {
		return nil
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

// MetadataIndex returns the index string for the first matcher from the Registry returning one
func (m *Matchers) MetadataIndex(event common.MapStr) string {
	m.RLock()
	defer m.RUnlock()
	for _, matcher := range m.matchers {
		index := matcher.MetadataIndex(event)
		if index != "" {
			return index
		}
	}

	// No index returned
	return ""
}

type GenDefaultMeta struct {
	annotations []string
	labels      []string
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
			Index: pod.Metadata.Name,
			Data:  data,
		},
	}
}

func (p *PodNameIndexer) GetIndexes(pod *Pod) []string {
	return []string{pod.Metadata.Name}
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
	containers := c.GetIndexes(pod)
	var metadata []MetadataIndex
	for i := 0; i < len(containers); i++ {
		containerMeta := commonMeta.Clone()
		containerMeta["container"] = common.MapStr{
			"name": pod.Status.ContainerStatuses[i].Name,
		}
		metadata = append(metadata, MetadataIndex{
			Index: containers[i],
			Data:  containerMeta,
		})
	}

	return metadata
}

func (c *ContainerIndexer) GetIndexes(pod *Pod) []string {
	var containers []string
	for _, status := range pod.Status.ContainerStatuses {
		cID := status.ContainerID
		if cID != "" {
			parts := strings.Split(cID, "//")
			if len(parts) == 2 {
				containers = append(containers, parts[1])
			}
		}
	}
	return containers
}

type FieldMatcher struct {
	MatchFields []string
}

func NewFieldMatcher(cfg common.Config) (Matcher, error) {
	config := struct {
		LookupFields []string `config:"lookup_fields"`
	}{}

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the `lookup_fields` configuration: %s", err)
	}

	if len(config.LookupFields) == 0 {
		return nil, fmt.Errorf("lookup_fields can not be empty")
	}

	return &FieldMatcher{MatchFields: config.LookupFields}, nil
}

func (f *FieldMatcher) MetadataIndex(event common.MapStr) string {
	for _, field := range f.MatchFields {
		keyIface, err := event.GetValue(field)
		if err == nil {
			key, ok := keyIface.(string)
			if ok {
				return key
			}
		}
	}

	return ""
}
