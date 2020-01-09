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

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/kubernetes"
)

type namespace struct {
	store    cache.Store
	resource *resource
}

func NewNamespaceMetadataGenerator(cfg *common.Config, namespaces cache.Store) MetaGen {
	no := &namespace{
		resource: NewResourceMetadataGenerator(cfg),
		store:    namespaces,
	}

	return no
}

func (n *namespace) Generate(obj kubernetes.Resource, opts ...FieldOptions) common.MapStr {
	_, ok := obj.(*kubernetes.Namespace)
	if !ok {
		return nil
	}

	meta := n.resource.Generate(obj, opts...)
	// TODO: Add extra fields in here if need be
	return meta
}

func (n *namespace) GenerateFromName(name string, opts ...FieldOptions) common.MapStr {
	if n.store == nil {
		return nil
	}

	if obj, ok, _ := n.store.GetByKey(name); ok {
		no, ok := obj.(*kubernetes.Namespace)
		if !ok {
			return nil
		}

		return n.Generate(no, opts...)
	} else {
		return nil
	}
}
