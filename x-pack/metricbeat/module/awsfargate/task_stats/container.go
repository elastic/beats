// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package task_stats

import (
	"github.com/elastic/beats/v8/libbeat/common"
	helpers "github.com/elastic/beats/v8/libbeat/common/docker"
)

// container is a struct representation of a container
type container struct {
	DockerId string
	Name     string
	Image    string
	Labels   map[string]string
}

// ContainerMetadata is an struct represents container metadata
type ContainerMetadata struct {
	Cluster   string
	TaskARN   string
	Family    string
	Revision  string
	Container *container
}

func getContainerStats(c *container) *container {
	return &container{
		DockerId: c.DockerId,
		Image:    c.Image,
		Name:     helpers.ExtractContainerName([]string{c.Name}),
		Labels:   deDotLabels(c.Labels),
	}
}

func deDotLabels(labels map[string]string) map[string]string {
	outputLabels := map[string]string{}
	for k, v := range labels {
		// This is necessary so that ES does not interpret '.' fields as new
		// nested JSON objects, and also makes this compatible with ES 2.x.
		label := common.DeDot(k)
		outputLabels[label] = v
	}

	return outputLabels
}
