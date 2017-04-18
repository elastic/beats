package kubernetes

import (
	"fmt"
	"strings"
	"sync"

	"github.com/elastic/beats/libbeat/common"

	corev1 "github.com/ericchiang/k8s/api/v1"
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
	GetMetadata(pod *corev1.Pod) []MetadataIndex

	// GetIndexes return the list of indexes the given pod belongs to. This function
	// must return the same indexes than GetMetadata
	GetIndexes(pod *corev1.Pod) []string
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
	indexers map[string]IndexConstructor
	matchers map[string]MatcherConstructor

	defaultIndexers []Indexer
	defaultMatchers []Matcher
}

type IndexConstructor func(config common.Config) (Indexer, error)
type MatcherConstructor func(config common.Config) (Matcher, error)

// NewRegister creates and returns a new Register.
func NewRegister() *Register {
	return &Register{
		indexers: make(map[string]IndexConstructor, 0),
		matchers: make(map[string]MatcherConstructor, 0),

		defaultIndexers: make([]Indexer, 0),
		defaultMatchers: make([]Matcher, 0),
	}
}

// AddIndexer to the register
func (r *Register) AddIndexer(name string, indexer IndexConstructor) {
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
func (r *Register) AddDefaultIndexer(indexer Indexer) {
	r.defaultIndexers = append(r.defaultIndexers, indexer)
}

// AddMatcher to the register
func (r *Register) AddDefaultMatcher(matcher Matcher) {
	r.defaultMatchers = append(r.defaultMatchers, matcher)
}

// AddIndexer to the register
func (r *Register) GetIndexer(name string) IndexConstructor {
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
func (i *Indexers) GetMetadata(pod *corev1.Pod) []MetadataIndex {
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
func (i *Indexers) GetIndexes(pod *corev1.Pod) []string {
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

// GenMetadata generates default metadata for the given pod
func GenMetadata(pod *corev1.Pod) common.MapStr {
	labelMap := common.MapStr{}
	for k, v := range pod.Metadata.Labels {
		labelMap[k] = v
	}
	return common.MapStr{
		"pod":       pod.Metadata.GetName(),
		"namespace": pod.Metadata.GetNamespace(),
		"labels":    labelMap,
	}
}

// PodNameIndexer implements default indexer based on pod name
type PodNameIndexer struct{}

func NewPodNameIndexer(_ common.Config) (Indexer, error) {
	return &PodNameIndexer{}, nil
}

func (p *PodNameIndexer) GetMetadata(pod *corev1.Pod) []MetadataIndex {
	data := GenMetadata(pod)
	return []MetadataIndex{
		{
			Index: pod.Metadata.GetName(),
			Data:  data,
		},
	}
}

func (p *PodNameIndexer) GetIndexes(pod *corev1.Pod) []string {
	return []string{pod.Metadata.GetName()}
}

// ContainerIndexer indexes pods based on all their containers IDs
type ContainerIndexer struct{}

func NewContainerIndexer(_ common.Config) (Indexer, error) {
	return &ContainerIndexer{}, nil
}

func (c *ContainerIndexer) GetMetadata(pod *corev1.Pod) []MetadataIndex {
	commonMeta := GenMetadata(pod)
	containers := c.GetIndexes(pod)
	var metadata []MetadataIndex
	for i := 0; i < len(containers); i++ {
		containerMeta := commonMeta.Clone()
		containerMeta["container"] = pod.Status.ContainerStatuses[i].Name
		metadata = append(metadata, MetadataIndex{
			Index: containers[i],
			Data:  containerMeta,
		})
	}

	return metadata
}

func (c *ContainerIndexer) GetIndexes(pod *corev1.Pod) []string {
	var containers []string
	for _, status := range pod.Status.ContainerStatuses {
		cID := status.GetContainerID()
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
