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
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/common/safemapstr"
	"strings"
)

// MetaGen allows creation of metadata from either Kubernetes resources or their Resource names.
type MetaGen interface {
	// Generate generates metadata for a given resource
	Generate(kubernetes.Resource, ...FieldOptions) common.MapStr
	// GenerateFromName generates metadata for a given resource based on it's name
	GenerateFromName(string, ...FieldOptions) common.MapStr
}

// FieldOptions allows additional enrichment to be done on top of existing metadata
type FieldOptions func(common.MapStr)

// WithFields FieldOption allows adding specific fields into the generated metadata
func WithFields(key string, value interface{}) FieldOptions {
	return func(meta common.MapStr) {
		safemapstr.Put(meta, key, value)
	}
}

// WithLabels FieldOption allows adding labels under sub-resource(kind)
// example if kind=namespace namespace.labels key will be added
func WithLabels(kind string) FieldOptions {
	return func(meta common.MapStr) {
		safemapstr.Put(meta, strings.ToLower(kind)+".labels", meta["labels"])
	}
}
