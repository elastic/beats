// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package nomad

import (
	"testing"

	"github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/assert"

	"github.com/hashicorp/nomad/helper"
	"github.com/hashicorp/nomad/helper/uuid"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/nomad"
)

func newJob(jobID string) *Job {
	return &Job{
		ID:          helper.StringToPtr(jobID),
		Region:      helper.StringToPtr(api.GlobalRegion),
		Name:        helper.StringToPtr("my-job"),
		Type:        helper.StringToPtr(nomad.JobTypeService),
		Status:      helper.StringToPtr(nomad.TaskStateRunning),
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
								Tags: []string{"tag-a", "tag-b", "${NOMAD_JOB_NAME}"},
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
		Namespace: api.DefaultNamespace,
	}

	config, err := common.NewConfigFrom(map[string]interface{}{
		"labels.dedot":        false,
		"annotations.dedot":   false,
		"include_annotations": []string{"b", "b.key"},
	})

	metaGen, err := NewMetaGenerator(config, nil)
	if err != nil {
		t.Fatal(err)
	}

	meta := metaGen.ResourceMetadata(alloc)
	tasks := metaGen.GroupMeta(alloc.Job)

	assert.Equal(t, "my-job", meta["job"])
	assert.Equal(t, "task-value", tasks[0]["key1"])
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

	metaGen, err := NewMetaGenerator(config, nil)
	if err != nil {
		t.Fatal(err)
	}

	tasks := metaGen.GroupMeta(alloc.Job)

	// verify that key1 is not included in the tasks metadata
	exists, err := tasks[0].HasKey("key1")
	assert.Nil(t, err)
	assert.False(t, exists)
}

func TestCronJob(t *testing.T) {
	jobID, allocID := uuid.Generate(), uuid.Generate()

	cron := &api.Job{
		ID:          helper.StringToPtr(jobID),
		Region:      helper.StringToPtr("global"),
		Name:        helper.StringToPtr("my-job"),
		Type:        helper.StringToPtr(nomad.JobTypeBatch),
		Status:      helper.StringToPtr(nomad.TaskStateRunning),
		Datacenters: []string{"europe-west4"},
		TaskGroups: []*TaskGroup{
			{
				Name: helper.StringToPtr("group"),
				Tasks: []*api.Task{
					{
						Name:   "web",
						Driver: "docker",
					},
					{
						Name:   "api",
						Driver: "docker",
					},
				},
			},
		},
		Periodic: &api.PeriodicConfig{
			SpecType: helper.StringToPtr(api.PeriodicSpecCron),
			Enabled:  helper.BoolToPtr(true),
		},
	}

	alloc := Resource{
		ID:        allocID,
		Job:       cron,
		Name:      "cronjob",
		Namespace: api.DefaultNamespace,
	}

	config, err := common.NewConfigFrom(map[string]interface{}{})

	metaGen, err := NewMetaGenerator(config, nil)
	if err != nil {
		t.Fatal(err)
	}

	meta := metaGen.ResourceMetadata(alloc)
	tasks := metaGen.GroupMeta(alloc.Job)

	assert.Equal(t, meta["alloc_id"], allocID)
	assert.Equal(t, meta["type"], nomad.JobTypeBatch)
	assert.Len(t, tasks, 2)
}
