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

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/mime"
	"github.com/menderesk/beats/v7/libbeat/processors"
	"github.com/menderesk/beats/v7/libbeat/processors/checks"
)

func init() {
	processors.RegisterPlugin("detect_mime_type",
		checks.ConfigChecked(NewDetectMimeType,
			checks.RequireFields("field", "target"),
			checks.AllowedFields("field", "target")))
}

type mimeTypeProcessor struct {
	Field  string `config:"field"`
	Target string `config:"target"`
}

// NewDetectMimeType constructs a new mime processor.
func NewDetectMimeType(cfg *common.Config) (processors.Processor, error) {
	mimeType := &mimeTypeProcessor{}
	if err := cfg.Unpack(mimeType); err != nil {
		return nil, errors.Wrapf(err, "fail to unpack the detect_mime_type configuration")
	}

	return mimeType, nil
}

func (m *mimeTypeProcessor) Run(event *beat.Event) (*beat.Event, error) {
	valI, err := event.GetValue(m.Field)
	if err != nil {
		// doesn't have the required field value to analyze
		return event, nil
	}
	val, _ := valI.(string)
	if val == "" {
		// wrong type or not set
		return event, nil
	}
	if mimeType := mime.Detect(val); mimeType != "" {
		_, err = event.PutValue(m.Target, mimeType)
	}
	return event, err
}

func (m *mimeTypeProcessor) String() string {
	return fmt.Sprintf("detect_mime_type=%+v->%+v", m.Field, m.Target)
}
