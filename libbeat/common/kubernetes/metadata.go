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

package kubernetes

import (
	"strings"

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
	IncludeLabels          []string `config:"include_labels"`
	ExcludeLabels          []string `config:"exclude_labels"`
	IncludeAnnotations     []string `config:"include_annotations"`
	IncludePodUID          bool     `config:"include_pod_uid"`
	IncludeCreatorMetadata bool     `config:"include_creator_metadata"`
}

// NewMetaGenerator initializes and returns a new kubernetes metadata generator
func NewMetaGenerator(cfg *common.Config) (MetaGenerator, error) {
	// default settings:
	generator := metaGenerator{
		IncludeCreatorMetadata: true,
	}

	err := cfg.Unpack(&generator)
	return &generator, err
}

// PodMetadata generates metadata for the given pod taking to account certain filters
func (g *metaGenerator) PodMetadata(pod *Pod) common.MapStr {
	labelMap := common.MapStr{}
	if len(g.IncludeLabels) == 0 {
		for k, v := range pod.Metadata.Labels {
			safemapstr.Put(labelMap, k, v)
		}
	} else {
		labelMap = generateMapSubset(pod.Metadata.Labels, g.IncludeLabels)
	}

	// Exclude any labels that are present in the exclude_labels config
	for _, label := range g.ExcludeLabels {
		delete(labelMap, label)
	}

	annotationsMap := generateMapSubset(pod.Metadata.Annotations, g.IncludeAnnotations)
	meta := common.MapStr{
		"pod": common.MapStr{
			"name": pod.Metadata.GetName(),
		},
		"node": common.MapStr{
			"name": pod.Spec.GetNodeName(),
		},
		"namespace": pod.Metadata.GetNamespace(),
	}

	// Add Pod UID metadata if enabled
	if g.IncludePodUID {
		safemapstr.Put(meta, "pod.uid", pod.Metadata.GetUid())
	}

	// Add controller metadata if present
	if g.IncludeCreatorMetadata {
		for _, ref := range pod.Metadata.OwnerReferences {
			if ref.GetController() {
				switch ref.GetKind() {
				// TODO grow this list as we keep adding more `state_*` metricsets
				case "Deployment",
					"ReplicaSet",
					"StatefulSet":
					safemapstr.Put(meta, strings.ToLower(ref.GetKind())+".name", ref.GetName())
				}
			}
		}
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
