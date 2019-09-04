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

package nomad

import (
	"fmt"
	"testing"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/hashicorp/nomad/helper"
	"github.com/hashicorp/nomad/helper/uuid"
)

func newJob(jobID string) *Job {
	return &Job{
		ID:          helper.StringToPtr(jobID),
		Region:      helper.StringToPtr("global"),
		Name:        helper.StringToPtr("my-job"),
		Type:        helper.StringToPtr(structs.JobTypeService),
		Datacenters: []string{"europe-west4"},
		Meta: map[string]string{
			"key1":    "job-value",
			"job-key": "job.value",
		},
		TaskGroups: []*TaskGroup{
			{
				Name: helper.StringToPtr("web"),
				Meta: map[string]string{
					"key1":      "group-value",
					"group-key": "group.value",
				},
				Tasks: []*api.Task{
					{
						Name: "task1",
						Meta: map[string]string{
							"key1":     "task-value",
							"task-key": "task.value",
						},
						Services: []*api.Service{
							{
								Id:   "service-a",
								Name: "web",
								Tags: []string{"tag-a", "tag-b"},
							},
							{
								Id:   "service-b",
								Name: "nginx",
								Tags: []string{"tag-c", "tag-d"},
							},
						},
					},
				},
			},
		},
	}
}

func TestAllocationMetadata(t *testing.T) {
	jobID := uuid.Generate()

	alloc := Resource{
		ID:        uuid.Generate(),
		Job:       newJob(jobID),
		Name:      "job.task",
		Namespace: "default",
	}

	config, err := common.NewConfigFrom(map[string]interface{}{
		"labels.dedot":        false,
		"annotations.dedot":   false,
		"include_annotations": []string{"b", "b.key"},
	})

	metaGen, err := NewMetaGenerator(config)
	if err != nil {
		t.Fatal(err)
	}

	meta := metaGen.ResourceMetadata(alloc)
	tasks, _ := meta["meta"].([]common.MapStr)
	flat := tasks[0].Flatten()

	fmt.Printf("%+v\n", meta)

	assert.Equal(t, "my-job", meta["job"])
	assert.Equal(t, "task-value", flat["task1.key1"])
	assert.Equal(t, 1, len(tasks))
	assert.Equal(t, []string{"europe-west4"}, meta["datacenters"])
}

func TestExcludeMetadata(t *testing.T) {
	jobID := uuid.Generate()

	alloc := Resource{
		ID:        uuid.Generate(),
		Job:       newJob(jobID),
		Name:      "job.task",
		Namespace: "default",
	}

	config, err := common.NewConfigFrom(map[string]interface{}{
		"exclude_labels": []string{"key1", "canary_tags"},
	})

	metaGen, err := NewMetaGenerator(config)
	if err != nil {
		t.Fatal(err)
	}

	meta := metaGen.ResourceMetadata(alloc)
	tasks, _ := meta["meta"].([]common.MapStr)
	flat := tasks[0].Flatten()

	exists, err := flat.HasKey("task1.key1")

	assert.NotNil(t, err)
	assert.False(t, exists)
}
