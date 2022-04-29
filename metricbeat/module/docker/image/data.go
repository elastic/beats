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

package image

import (
	"time"

	"github.com/docker/docker/api/types"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/docker"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func eventsMapping(imagesList []types.ImageSummary, dedot bool) []mapstr.M {
	events := []mapstr.M{}
	for _, image := range imagesList {
		events = append(events, eventMapping(&image, dedot))
	}
	return events
}

func eventMapping(image *types.ImageSummary, dedot bool) mapstr.M {
	event := mapstr.M{
		"id": mapstr.M{
			"current": image.ID,
			"parent":  image.ParentID,
		},
		"created": common.Time(time.Unix(image.Created, 0)),
		"size": mapstr.M{
			"regular": image.Size,
			"virtual": image.VirtualSize,
		},
		"tags": image.RepoTags,
	}
	if len(image.Labels) > 0 {
		labels := docker.DeDotLabels(image.Labels, dedot)
		event["labels"] = labels
	}
	return event
}
