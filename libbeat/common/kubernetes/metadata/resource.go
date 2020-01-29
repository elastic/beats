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

package metadata

import (
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/kubernetes"
	"github.com/elastic/beats/libbeat/common/safemapstr"
)

// Resource generates metadata for any kubernetes resource
type Resource struct {
	config *Config
}

// NewResourceMetadataGenerator creates a metadata generator for a generic resource
func NewResourceMetadataGenerator(cfg *common.Config) *Resource {
	config := defaultConfig()
	config.Unmarshal(cfg)

	return &Resource{
		config: &config,
	}
}

// Generate takes a kind and an object and creates metadata for the same
func (r *Resource) Generate(kind string, obj kubernetes.Resource, options ...FieldOptions) common.MapStr {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return nil
	}

	labelMap := common.MapStr{}
	if len(r.config.IncludeLabels) == 0 {
		for k, v := range accessor.GetLabels() {
			if r.config.LabelsDedot {
				label := common.DeDot(k)
				labelMap.Put(label, v)
			} else {
				safemapstr.Put(labelMap, k, v)
			}
		}
	} else {
		labelMap = generateMapSubset(accessor.GetLabels(), r.config.IncludeLabels, r.config.LabelsDedot)
	}

	// Exclude any labels that are present in the exclude_labels config
	for _, label := range r.config.ExcludeLabels {
		labelMap.Delete(label)
	}

	annotationsMap := generateMapSubset(accessor.GetAnnotations(), r.config.IncludeAnnotations, r.config.AnnotationsDedot)

	meta := common.MapStr{
		strings.ToLower(kind): common.MapStr{
			"name": accessor.GetName(),
			"uid":  string(accessor.GetUID()),
		},
	}

	if accessor.GetNamespace() != "" {
		// TODO make this namespace.name in 8.0
		safemapstr.Put(meta, "namespace", accessor.GetNamespace())
	}

	// Add controller metadata if present
	if r.config.IncludeCreatorMetadata {
		for _, ref := range accessor.GetOwnerReferences() {
			if ref.Controller != nil && *ref.Controller {
				switch ref.Kind {
				// TODO grow this list as we keep adding more `state_*` metricsets
				case "Deployment",
					"ReplicaSet",
					"StatefulSet":
					safemapstr.Put(meta, strings.ToLower(ref.Kind)+".name", ref.Name)
				}
			}
		}
	}

	if len(labelMap) != 0 {
		safemapstr.Put(meta, strings.ToLower(kind)+".labels", labelMap)
	}

	if len(annotationsMap) != 0 {
		safemapstr.Put(meta, strings.ToLower(kind)+".annotations", annotationsMap)
	}

	for _, option := range options {
		option(meta)
	}

	return meta
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
