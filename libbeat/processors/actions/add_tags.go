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
	"strings"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	conf "github.com/elastic/elastic-agent-libs/config"
)

type addTags struct {
	tags   []string
	target string
}

func init() {
	processors.RegisterPlugin("add_tags",
		checks.ConfigChecked(createAddTags,
			checks.RequireFields("tags"),
			checks.AllowedFields("tags", "target", "when")))
}

func createAddTags(c *conf.C) (processors.Processor, error) {
	config := struct {
		Tags   []string `config:"tags" validate:"required"`
		Target string   `config:"target"`
	}{}

	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the add_tags configuration: %s", err)
	}

	return NewAddTags(config.Target, config.Tags), nil
}

// NewAddTags creates a new processor for adding tags to a field.
// If the target field already contains tags, then the new tags will be
// appended to the existing list of tags.
func NewAddTags(target string, tags []string) processors.Processor {
	if target == "" {
		target = common.TagsKey
	}

	// make sure capacity == length such that different processors adding more tags
	// do not change/overwrite each other on append
	if cap(tags) != len(tags) {
		tmp := make([]string, len(tags), len(tags))
		copy(tmp, tags)
		tags = tmp
	}

	return &addTags{tags: tags, target: target}
}

func (at *addTags) Run(event *beat.Event) (*beat.Event, error) {
	common.AddTagsWithKey(event.Fields, at.target, at.tags)
	return event, nil
}

func (at *addTags) String() string {
	return fmt.Sprintf("add_tags=%v", strings.Join(at.tags, ","))
}
