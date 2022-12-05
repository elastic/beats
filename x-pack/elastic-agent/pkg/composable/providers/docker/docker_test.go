// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/common/docker"
)

func TestGenerateData(t *testing.T) {
	container := &docker.Container{
		ID:   "abc",
		Name: "foobar",
		Labels: map[string]string{
			"do.not.include":          "true",
			"co.elastic.logs/disable": "true",
		},
	}
	event := bus.Event{
		"container": container,
	}

	data, err := generateData(event)
	require.NoError(t, err)
	mapping := map[string]interface{}{
		"container": map[string]interface{}{
			"id":    container.ID,
			"name":  container.Name,
			"image": container.Image,
			"labels": common.MapStr{
				"do": common.MapStr{"not": common.MapStr{"include": "true"}},
				"co": common.MapStr{"elastic": common.MapStr{"logs/disable": "true"}},
			},
		},
	}
	processors := []map[string]interface{}{
		{
			"add_fields": map[string]interface{}{
				"fields": map[string]interface{}{
					"id":    container.ID,
					"name":  container.Name,
					"image": container.Image,
					"labels": common.MapStr{
						"do_not_include":          "true",
						"co_elastic_logs/disable": "true",
					},
				},
				"target": "container",
			},
		},
	}

	assert.Equal(t, container, data.container)
	assert.Equal(t, mapping, data.mapping)
	assert.Equal(t, processors, data.processors)
}
