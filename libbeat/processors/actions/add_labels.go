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

package actions

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// LabelsKey is the default target key for the add_labels processor.
const LabelsKey = "labels"

func init() {
	processors.RegisterPlugin("add_labels",
		checks.ConfigChecked(createAddLabels,
			checks.RequireFields(LabelsKey),
			checks.AllowedFields(LabelsKey, "when")))
}

func createAddLabels(c *common.Config) (processors.Processor, error) {
	config := struct {
		Labels mapstr.M `config:"labels" validate:"required"`
	}{}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the add_fields configuration: %w", err)
	}

	flatLabels, err := flattenLabels(config.Labels)
	if err != nil {
		return nil, fmt.Errorf("failed to flatten labels: %w", err)
	}

	return makeFieldsProcessor(LabelsKey, flatLabels, true), nil
}

// NewAddLabels creates a new processor adding the given object to events. Set
// `shared` true if there is the chance of labels being changed/modified by
// subsequent processors.
// If labels contains nested objects, NewAddLabels will flatten keys into labels by
// by joining names with a dot ('.') .
// The labels will be inserted into the 'labels' field.
func NewAddLabels(labels mapstr.M, shared bool) (processors.Processor, error) {
	flatLabels, err := flattenLabels(labels)
	if err != nil {
		return nil, fmt.Errorf("failed to flatten labels: %w", err)
	}

	return NewAddFields(mapstr.M{
		LabelsKey: flatLabels,
	}, shared, true), nil
}

func flattenLabels(labels mapstr.M) (mapstr.M, error) {
	labelConfig, err := common.NewConfigFrom(labels)
	if err != nil {
		return nil, err
	}

	flatKeys := labelConfig.FlattenedKeys()
	flatMap := make(mapstr.M, len(flatKeys))
	for _, k := range flatKeys {
		v, err := labelConfig.String(k, -1)
		if err != nil {
			return nil, err
		}
		flatMap[k] = v
	}

	return flatMap, nil
}
