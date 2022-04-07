// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package nomad

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/common"
)

func newJob(jobID string) *Job {
	return &Job{
		ID:          StringToPtr(jobID),
		Region:      StringToPtr(api.GlobalRegion),
		Name:        StringToPtr("my-job"),
		Type:        StringToPtr(JobTypeService),
		Status:      StringToPtr(JobStatusRunning),
		Datacenters: []string{"europe-west4"},
		Meta: map[string]string{
			"key1":    "job-value",
			"job-key": "job.value",
		},
		TaskGroups: []*TaskGroup{
			{
				Name: StringToPtr("web"),
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
	jobID := newUUID()

	alloc := Resource{
		ID:        newUUID(),
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

	assert.EqualValues(t, common.MapStr{
		"name": "my-job",
		"type": "service",
	}, meta["job"])
	assert.Equal(t, "task-value", tasks[0]["key1"])
	assert.Equal(t, []string{"europe-west4"}, meta["datacenter"])
}

func TestExcludeMetadata(t *testing.T) {
	jobID := newUUID()

	alloc := Resource{
		ID:        newUUID(),
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
	jobID, allocID := newUUID(), newUUID()

	cron := &api.Job{
		ID:          StringToPtr(jobID),
		Region:      StringToPtr("global"),
		Name:        StringToPtr("my-job"),
		Type:        StringToPtr(JobTypeBatch),
		Status:      StringToPtr(JobStatusRunning),
		Datacenters: []string{"europe-west4"},
		TaskGroups: []*TaskGroup{
			{
				Name: StringToPtr("group"),
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
			SpecType: StringToPtr(api.PeriodicSpecCron),
			Enabled:  BoolToPtr(true),
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

	assert.EqualValues(t, common.MapStr{
		"name":   alloc.Name,
		"id":     allocID,
		"status": "",
	}, meta["allocation"])
	assert.EqualValues(t, common.MapStr{
		"name": *cron.Name,
		"type": JobTypeBatch,
	}, meta["job"])
	assert.Len(t, tasks, 2)
}

func newUUID() string {
	id, err := uuid.NewV4()

	if err != nil {
		return "b87daa1c-b091-4355-a2d2-60f9f3bff1b0"
	}

	return id.String()
}
