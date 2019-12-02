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

	"github.com/elastic/beats/libbeat/common/safemapstr"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/kubernetes"
)

type service struct {
	store     cache.Store
	namespace MetaGen
	resource  *resource
}

func NewServiceMetadataGenerator(cfg *common.Config, services cache.Store, namespace MetaGen) MetaGen {
	po := &service{
		resource:  NewResourceMetadataGenerator(cfg),
		store:     services,
		namespace: namespace,
	}

	return po
}

func (s *service) Generate(obj kubernetes.Resource, opts ...FieldOptions) common.MapStr {
	po, ok := obj.(*kubernetes.Pod)
	if !ok {
		return nil
	}

	out := s.resource.Generate(obj, opts...)

	if s.namespace != nil {
		meta := s.namespace.GenerateFromName(po.GetNamespace())
		safemapstr.Put(out, "namespace", meta)
	}

	return out
}

func (s *service) GenerateFromName(name string, opts ...FieldOptions) common.MapStr {
	if s.store == nil {
		return nil
	}

	if obj, ok, _ := s.store.GetByKey(name); ok {
		po, ok := obj.(*kubernetes.Pod)
		if !ok {
			return nil
		}

		return s.Generate(po, opts...)
	} else {
		return nil
	}
}
