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

package docker

import (
	"strings"

	"github.com/docker/docker/api/types"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/safemapstr"
)

type Container struct {
	ID     string
	Name   string
	Image  string
	Labels common.MapStr
}

func (c *Container) ToMapStr() common.MapStr {
	m := common.MapStr{
		"id":    c.ID,
		"name":  c.Name,
		"image": c.Image,
	}

	if len(c.Labels) > 0 {
		m["labels"] = c.Labels
	}
	return m
}

// NewContainer converts Docker API container to an internal structure, it applies
// dedot to container labels if dedot is true, or stores them in a nested way if it's
// false
func NewContainer(container *types.Container, dedot bool) *Container {
	return &Container{
		ID:     container.ID,
		Name:   ExtractContainerName(container.Names),
		Labels: DeDotLabels(container.Labels, dedot),
		Image:  container.Image,
	}
}

func ExtractContainerName(names []string) string {
	output := names[0]

	if len(names) > 1 {
		for _, name := range names {
			if strings.Count(output, "/") > strings.Count(name, "/") {
				output = name
			}
		}
	}
	return strings.Trim(output, "/")
}

// DeDotLabels returns a new common.MapStr containing a copy of the labels
// where the dots have been converted into nested structure, avoiding
// possible mapping errors
func DeDotLabels(labels map[string]string, dedot bool) common.MapStr {
	outputLabels := common.MapStr{}
	for k, v := range labels {
		if dedot {
			// This is necessary so that ES does not interpret '.' fields as new
			// nested JSON objects, and also makes this compatible with ES 2.x.
			label := common.DeDot(k)
			outputLabels.Put(label, v)
		} else {
			// If we don't dedot we ensure there are no mapping errors with safemapstr
			safemapstr.Put(outputLabels, k, v)
		}
	}

	return outputLabels
}
