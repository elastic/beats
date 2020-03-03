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
	"k8s.io/client-go/tools/cache"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
)

type pod struct {
	store     cache.Store
	node      MetaGen
	namespace MetaGen
	resource  *Resource
}

// NewPodMetadataGenerator creates a metagen for pod resources
func NewPodMetadataGenerator(cfg *common.Config, pods cache.Store, node MetaGen, namespace MetaGen) MetaGen {
	return &pod{
		resource:  NewResourceMetadataGenerator(cfg),
		store:     pods,
		node:      node,
		namespace: namespace,
	}
}

// Generate generates pod metadata from a resource object
func (p *pod) Generate(obj kubernetes.Resource, opts ...FieldOptions) common.MapStr {
	po, ok := obj.(*kubernetes.Pod)
	if !ok {
		return nil
	}

	out := p.resource.Generate("pod", obj, opts...)
	// TODO: remove this call when moving to 8.0
	out = p.exportPodLabelsAndAnnotations(out)

	if p.node != nil {
		meta := p.node.GenerateFromName(po.Spec.NodeName)
		if meta != nil {
			out.Put("node", meta["node"])
		} else {
			out.Put("node.name", po.Spec.NodeName)
		}
	} else {
		out.Put("node.name", po.Spec.NodeName)
	}

	if p.namespace != nil {
		meta := p.namespace.GenerateFromName(po.GetNamespace())
		if meta != nil {
			// Use this in 8.0
			//out.Put("namespace", meta["namespace"])
			out.DeepUpdate(meta)
		}
	}
	return out
}

// GenerateFromName generates pod metadata from a pod name
func (p *pod) GenerateFromName(name string, opts ...FieldOptions) common.MapStr {
	if p.store == nil {
		return nil
	}

	if obj, ok, _ := p.store.GetByKey(name); ok {
		po, ok := obj.(*kubernetes.Pod)
		if !ok {
			return nil
		}

		return p.Generate(po, opts...)
	}

	return nil
}

func (p *pod) exportPodLabelsAndAnnotations(in common.MapStr) common.MapStr {
	labels, err := in.GetValue("pod.labels")
	if err != nil {
		return in
	}
	in.Put("labels", labels)
	in.Delete("pod.labels")

	annotations, err := in.GetValue("pod.annotations")
	if err != nil {
		return in
	}
	in.Put("annotations", annotations)
	in.Delete("pod.annotations")

	return in
}
