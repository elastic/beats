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
	"fmt"
	"strings"

	"k8s.io/client-go/kubernetes"

	"github.com/elastic/beats/libbeat/logp"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/safemapstr"
)

// MetaGenerator builds metadata objects for pods and containers
type MetaGenerator interface {
	// ResourceMetadata generates metadata for the given kubernetes object taking to account certain filters
	ResourceMetadata(obj Resource) common.MapStr

	// PodMetadata generates metadata for the given pod taking to account certain filters
	PodMetadata(pod *Pod) common.MapStr

	// Containermetadata generates metadata for the given container of a pod
	ContainerMetadata(pod *Pod, container string, image string) common.MapStr
}

// MetaGeneratorConfig settings
type MetaGeneratorConfig struct {
	IncludeLabels      []string `config:"include_labels"`
	ExcludeLabels      []string `config:"exclude_labels"`
	IncludeAnnotations []string `config:"include_annotations"`

	LabelsDedot      bool `config:"labels.dedot"`
	AnnotationsDedot bool `config:"annotations.dedot"`

	// Undocumented settings, to be deprecated in favor of `drop_fields` processor:
	IncludeCreatorMetadata bool `config:"include_creator_metadata"`
	k8sClient              kubernetes.Interface
}

type metaGenerator = MetaGeneratorConfig

// DefaultMetaGeneratorConfig initializes and returns a new MetaGeneratorConfig with default values
func DefaultMetaGeneratorConfig(client kubernetes.Interface) MetaGeneratorConfig {
	return MetaGeneratorConfig{
		IncludeCreatorMetadata: true,
		LabelsDedot:            true,
		AnnotationsDedot:       true,
		k8sClient:              client,
	}
}

// NewMetaGenerator initializes and returns a new kubernetes metadata generator
func NewMetaGenerator(cfg *common.Config, client kubernetes.Interface) (MetaGenerator, error) {
	generator := DefaultMetaGeneratorConfig(client)

	err := cfg.Unpack(&generator)
	return &generator, err
}

// NewMetaGeneratorFromConfig initializes and returns a new kubernetes metadata generator
func NewMetaGeneratorFromConfig(cfg *MetaGeneratorConfig) MetaGenerator {
	return cfg
}

// ResourceMetadata generates metadata for the given kubernetes object taking to account certain filters
func (g *metaGenerator) ResourceMetadata(obj Resource) common.MapStr {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return nil
	}

	labelMap := common.MapStr{}
	if len(g.IncludeLabels) == 0 {
		for k, v := range accessor.GetLabels() {
			if g.LabelsDedot {
				label := common.DeDot(k)
				labelMap.Put(label, v)
			} else {
				safemapstr.Put(labelMap, k, v)
			}
		}
	} else {
		labelMap = generateMapSubset(accessor.GetLabels(), g.IncludeLabels, g.LabelsDedot)
	}

	// Exclude any labels that are present in the exclude_labels config
	for _, label := range g.ExcludeLabels {
		labelMap.Delete(label)
	}

	annotationsMap := generateMapSubset(accessor.GetAnnotations(), g.IncludeAnnotations, g.AnnotationsDedot)
	metadata := common.MapStr{}
	if accessor.GetNamespace() != "" {
		namespase := accessor.GetNamespace()
		metadata["namespace"] = accessor.GetNamespace()

		ns, err := g.k8sClient.CoreV1().Namespaces().Get(namespase, metav1.GetOptions{})
		if err != nil {
			logp.Error(err)
		}

		namespaceaccessor, err := meta.Accessor(ns)

		if err != nil {
			return nil
		}
		namespacelabels := namespaceaccessor.GetLabels()
		
		for k, v := range namespacelabels {
			if g.LabelsDedot {
				label := common.DeDot(k)
				metakey := fmt.Sprintf("namespace_%v", label)
				metadata.Put(metakey, v)
			} else {
				metakey := fmt.Sprintf("namespace_%v", k)
				safemapstr.Put(labelMap, metakey, v)
			}
		}
	}

	// Add controller metadata if present
	if g.IncludeCreatorMetadata {
		for _, ref := range accessor.GetOwnerReferences() {
			if *ref.Controller {
				switch ref.Kind {
				// TODO grow this list as we keep adding more `state_*` metricsets
				case "Deployment",
					"ReplicaSet",
					"StatefulSet":
					safemapstr.Put(metadata, strings.ToLower(ref.Kind)+".name", ref.Name)
				}
			}
		}
	}

	if len(labelMap) != 0 {
		metadata["labels"] = labelMap
	}

	if len(annotationsMap) != 0 {
		metadata["annotations"] = annotationsMap
	}

	return metadata
}

// PodMetadata generates metadata for the given pod taking to account certain filters
func (g *metaGenerator) PodMetadata(pod *Pod) common.MapStr {
	podMeta := g.ResourceMetadata(pod)
	safemapstr.Put(podMeta, "pod.uid", string(pod.GetObjectMeta().GetUID()))
	safemapstr.Put(podMeta, "pod.name", pod.GetObjectMeta().GetName())
	safemapstr.Put(podMeta, "node.name", pod.Spec.NodeName)

	return podMeta
}

// Containermetadata generates metadata for the given container of a pod
func (g *metaGenerator) ContainerMetadata(pod *Pod, container string, image string) common.MapStr {
	podMeta := g.PodMetadata(pod)

	// Add container details
	podMeta["container"] = common.MapStr{
		"name":  container,
		"image": image,
	}

	return podMeta
}

func generateMapSubset(input map[string]string, keys []string, dedot bool) common.MapStr {
	output := common.MapStr{}
	if input == nil {
		return output
	}

	for _, key := range keys {
		value, ok := input[key]
		if ok {
			if dedot {
				dedotKey := common.DeDot(key)
				output.Put(dedotKey, value)
			} else {
				safemapstr.Put(output, key, value)
			}
		}
	}

	return output
}
