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

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/processors/checks"
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
		Labels common.MapStr `config:"labels" validate:"required"`
	}{}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the add_fields configuration: %s", err)
	}

	return makeFieldsProcessor(LabelsKey, config.Labels.Flatten(), true), nil
}

// NewAddLabels creates a new processor adding the given object to events. Set
// `shared` true if there is the chance of labels being changed/modified by
// subsequent processors.
// If labels contains nested objects, NewAddLabels will flatten keys into labels by
// by joining names with a dot ('.') .
// The labels will be inserted into the 'labels' field.
func NewAddLabels(labels common.MapStr, shared bool) processors.Processor {
	return NewAddFields(common.MapStr{
		LabelsKey: labels.Flatten(),
	}, shared, true)
}
