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

package add_ben_metadata

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/pkg/errors"
)

type addBenMetadata struct {
	unitandcomputer string
}

type testEvent struct {
	name string
}

const processorName = "add_ben_metadata"

// New constructs a new Add ID processor.
func Newc(c *common.Config) (processors.Processor, error) {
	config := struct {
		Format string `config:"format"`
	}{
		Format: "offset",
	}

	err := c.Unpack(&config)
	if err != nil {
		return nil, errors.Wrap(err, "fail to unpack the add_locale configuration")
	}

	p := &testEvent{
		name: "ben",
	}

	return p, nil
}

func (p *testEvent) Run(event *beat.Event) (*beat.Event, error) {
	p.name = "ben"
	event.PutValue("dadada", p.name)
	return event, nil
}
