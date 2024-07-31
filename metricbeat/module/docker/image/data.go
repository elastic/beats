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

	"github.com/docker/docker/api/types/image"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/docker"
)

<<<<<<< HEAD
func eventsMapping(imagesList []types.ImageSummary, dedot bool) []common.MapStr {
	events := []common.MapStr{}
=======
func eventsMapping(imagesList []image.Summary, dedot bool) []mapstr.M {
	events := []mapstr.M{}
>>>>>>> 3c65545078 (Upgrad elastic-agent-system-metrics to v0.10.7. (#40397))
	for _, image := range imagesList {
		events = append(events, eventMapping(&image, dedot))
	}
	return events
}

<<<<<<< HEAD
func eventMapping(image *types.ImageSummary, dedot bool) common.MapStr {
	event := common.MapStr{
		"id": common.MapStr{
			"current": image.ID,
			"parent":  image.ParentID,
		},
		"created": common.Time(time.Unix(image.Created, 0)),
		"size": common.MapStr{
			"regular": image.Size,
			"virtual": image.VirtualSize,
=======
func eventMapping(img *image.Summary, dedot bool) mapstr.M {
	event := mapstr.M{
		"id": mapstr.M{
			"current": img.ID,
			"parent":  img.ParentID,
		},
		"created": common.Time(time.Unix(img.Created, 0)),
		"size": mapstr.M{
			"regular": img.Size,
			"virtual": img.VirtualSize,
>>>>>>> 3c65545078 (Upgrad elastic-agent-system-metrics to v0.10.7. (#40397))
		},
		"tags": img.RepoTags,
	}
	if len(img.Labels) > 0 {
		labels := docker.DeDotLabels(img.Labels, dedot)
		event["labels"] = labels
	}
	return event
}
