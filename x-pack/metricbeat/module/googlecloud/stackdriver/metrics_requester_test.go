// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stackdriver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringInSlice(t *testing.T) {
	cases := []struct {
		title          string
		m              string
		region         string
		zone           string
		expectedFilter string
	}{
		{
			"construct filter with zone",
			"compute.googleapis.com/instance/cpu/utilization",
			"",
			"us-east1-b",
			"metric.type=\"compute.googleapis.com/instance/cpu/utilization\" AND resource.labels.zone = \"us-east1-b\"",
		},
		{
			"construct filter with region",
			"compute.googleapis.com/instance/cpu/utilization",
			"us-east1",
			"",
			"metric.type=\"compute.googleapis.com/instance/cpu/utilization\" AND resource.labels.zone = starts_with(\"us-east1\")",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			filter := constructFilter(c.m, c.region, c.zone)
			assert.Equal(t, c.expectedFilter, filter)
		})
	}
}
