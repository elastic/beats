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
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

type addLabels struct {
	labels common.MapStr
	shared bool
}

// LabelsKey is the default target key for the add_labels processor.
const LabelsKey = "labels"

func init() {
	processors.RegisterPlugin("add_labels",
		configChecked(createAddLabels,
			requireFields("labels"),
			allowedFields("labels", "target", "when")))
}

func createAddLabels(c *common.Config) (processors.Processor, error) {
	config := struct {
		Labels common.MapStr `config:"labels" validate:"required"`
		Target *string       `config:"target"`
	}{}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the add_fields configuration: %s", err)
	}

	var target string
	if config.Target == nil {
		target = LabelsKey
	} else {
		target = *config.Target
	}

	labels := config.Labels
	if target != "" {
		labels = common.MapStr{
			target: labels,
		}
	}

	return NewAddLabels(labels, true), nil
}

// NewAddLabels creates a new processor adding the given object to events. Set
// `shared` true if there is the chance of labels being changed/modified by
// subsequent processors.
func NewAddLabels(labels common.MapStr, shared bool) processors.Processor {
	return &addLabels{labels: labels, shared: shared}
}

func (af *addLabels) Run(event *beat.Event) (*beat.Event, error) {
	labels := af.labels
	if af.shared {
		labels = labels.Clone()
	}

	event.Fields.DeepUpdate(labels)
	return event, nil
}

func (af *addLabels) String() string {
	s, _ := json.Marshal(af.labels)
	return fmt.Sprintf("add_labels=%s", s)
}
