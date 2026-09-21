// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package nomad

import (
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func newJob(jobID string) *Job {
	return &Job{
		ID:          new(jobID),
		Region:      new(api.GlobalRegion),
		Name:        new("my-job"),
		Type:        new(JobTypeService),
		Status:      new(JobStatusRunning),
		Datacenters: []string{"europe-west4"},
		Meta: map[string]string{
			"key1":    "job-value",
			"job-key": "job.value",
		},
		TaskGroups: []*TaskGroup{
			{
				Name: new("web"),
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
								Name: "web",
								Tags: []string{"tag-a", "tag-b", "${NOMAD_JOB_NAME}"},
							},
							{
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

	config, err := conf.NewConfigFrom(map[string]any{
		"labels.dedot":        false,
		"annotations.dedot":   false,
		"include_annotations": []string{"b", "b.key"},
	})
	require.NoError(t, err)

	metaGen, err := NewMetaGenerator(config, nil)
	if err != nil {
		t.Fatal(err)
	}

	meta := metaGen.ResourceMetadata(alloc)
	tasks := metaGen.GroupMeta(alloc.Job)

	assert.EqualValues(t, mapstr.M{
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

	config, err := conf.NewConfigFrom(map[string]any{
		"exclude_labels": []string{"key1", "canary_tags"},
	})
	require.NoError(t, err)

	metaGen, err := NewMetaGenerator(config, nil)
	if err != nil {
		t.Fatal(err)
	}

	tasks := metaGen.GroupMeta(alloc.Job)

	// verify that key1 is not included in the tasks metadata
	exists, err := tasks[0].HasKey("key1")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCronJob(t *testing.T) {
	jobID, allocID := newUUID(), newUUID()

	cron := &api.Job{
		ID:          new(jobID),
		Region:      new("global"),
		Name:        new("my-job"),
		Type:        new(JobTypeBatch),
		Status:      new(JobStatusRunning),
		Datacenters: []string{"europe-west4"},
		TaskGroups: []*TaskGroup{
			{
				Name: new("group"),
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
			SpecType: new(api.PeriodicSpecCron),
			Enabled:  new(true),
		},
	}

	alloc := Resource{
		ID:        allocID,
		Job:       cron,
		Name:      "cronjob",
		Namespace: api.DefaultNamespace,
	}

	config, err := conf.NewConfigFrom(map[string]any{})
	require.NoError(t, err)

	metaGen, err := NewMetaGenerator(config, nil)
	if err != nil {
		t.Fatal(err)
	}

	meta := metaGen.ResourceMetadata(alloc)
	tasks := metaGen.GroupMeta(alloc.Job)

	assert.EqualValues(t, mapstr.M{
		"name":   alloc.Name,
		"id":     allocID,
		"status": "",
	}, meta["allocation"])
	assert.EqualValues(t, mapstr.M{
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
