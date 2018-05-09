package kubernetes

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/safemapstr"
)

// MetaGenerator builds metadata objects for pods and containers
type MetaGenerator interface {
	// PodMetadata generates metadata for the given pod taking to account certain filters
	PodMetadata(pod *Pod) common.MapStr

	// Containermetadata generates metadata for the given container of a pod
	ContainerMetadata(pod *Pod, container string) common.MapStr
}

type metaGenerator struct {
	annotations   []string
	labels        []string
	labelsExclude []string
}

// NewMetaGenerator initializes and returns a new kubernetes metadata generator
func NewMetaGenerator(annotations, labels, labelsExclude []string) MetaGenerator {
	return &metaGenerator{
		annotations:   annotations,
		labels:        labels,
		labelsExclude: labelsExclude,
	}
}

// PodMetadata generates metadata for the given pod taking to account certain filters
func (g *metaGenerator) PodMetadata(pod *Pod) common.MapStr {
	labelMap := common.MapStr{}
	if len(g.labels) == 0 {
		for k, v := range pod.Metadata.Labels {
			safemapstr.Put(labelMap, k, v)
		}
	} else {
		labelMap = generateMapSubset(pod.Metadata.Labels, g.labels)
	}

	// Exclude any labels that are present in the exclude_labels config
	for _, label := range g.labelsExclude {
		delete(labelMap, label)
	}

	annotationsMap := generateMapSubset(pod.Metadata.Annotations, g.annotations)
	meta := common.MapStr{
		"pod": common.MapStr{
			"name": pod.Metadata.Name,
		},
		"node": common.MapStr{
			"name": pod.Spec.NodeName,
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

// Containermetadata generates metadata for the given container of a pod
func (g *metaGenerator) ContainerMetadata(pod *Pod, container string) common.MapStr {
	podMeta := g.PodMetadata(pod)

	// Add container details
	podMeta["container"] = common.MapStr{
		"name": container,
	}

	return podMeta
}

func generateMapSubset(input map[string]string, keys []string) common.MapStr {
	output := common.MapStr{}
	if input == nil {
		return output
	}

	for _, key := range keys {
		value, ok := input[key]
		if ok {
			safemapstr.Put(output, key, value)
		}
	}

	return output
}
