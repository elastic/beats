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

package events

import "github.com/elastic/beats/v7/libbeat/beat"

const (
	FieldMetaID       = "_id"
	FieldMetaIndex    = "index"
	FieldMetaRawIndex = "raw_index"
	FieldMetaAlias    = "alias"
	FieldMetaPipeline = "pipeline"

	// FieldMetaOpType defines the metadata key name for event operation type.
	// The key's value can be an empty string, `create`, `index`, or `delete`. If empty, it will assume
	// either `create` or `index`. See `createEventBulkMeta`. If in doubt, set explicitly.
	FieldMetaOpType = "op_type"

	FieldMetaOpTypeCreate MetaOpType = iota
	FieldMetaOpTypeDelete
	FieldMetaOpTypeIndex
)

type MetaOpType int

func (o MetaOpType) String() string {
	return []string{"create", "delete", "index"}[o]
}

// GetMetaStringValue returns the value of the given event metadata string field
func GetMetaStringValue(e beat.Event, key string) (string, error) {
	tmp, err := e.Meta.GetValue(key)
	if err != nil {
		return "", err
	}

	if s, ok := tmp.(string); ok {
		return s, nil
	}

	return "", nil
}
