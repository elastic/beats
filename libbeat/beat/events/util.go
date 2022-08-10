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
	// FieldMetaID defines the ID for the event. Also see FieldMetaOpType.
	FieldMetaID = "_id"

	// FieldMetaAlias defines the index alias to use for the event. If set, it takes
	// precedence over values defined using FieldMetaIndex or FieldMetaRawIndex.
	FieldMetaAlias = "alias"

	// FieldMetaIndex defines the data stream name to use for the event.
	// If set, it takes precedence over the value defined using FieldMetaRawIndex.
	FieldMetaIndex = "index"

	// FieldMetaRawIndex defines the raw index name to use for the event. It is used as-is, without
	// any additional manipulation.
	FieldMetaRawIndex = "raw_index"

	// FieldMetaPipeline defines the ingest node pipeline to use for this event.
	FieldMetaPipeline = "pipeline"

	// FieldMetaOpType defines the metadata key name for event operation type to use with the Elasticsearch
	// Bulk API encoding of the event. The key's value can be an empty string, `create`, `index`, or `delete`.
	// If empty, `create` will be used if FieldMetaID is set; otherwise `index` will be used.
	FieldMetaOpType = "op_type"
)

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

// GetOpType returns the event's op_type, if set
func GetOpType(e beat.Event) OpType {
	tmp, err := e.Meta.GetValue(FieldMetaOpType)
	if err != nil {
		return OpTypeDefault
	}

	switch v := tmp.(type) {
	case OpType:
		return v
	case string:
		switch v {
		case "create":
			return OpTypeCreate
		case "index":
			return OpTypeIndex
		case "delete":
			return OpTypeDelete
		}
	}

	return OpTypeDefault
}
