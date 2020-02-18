// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stackdriver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFilterForMetric(t *testing.T) {
	r := stackdriverMetricsRequester{config: config{Zone: "us-central1-a"}}

	s := r.getFilterForMetric("compute.googleapis.com/firewall/dropped_bytes_count")
	assert.Equal(t, `metric.type="compute.googleapis.com/firewall/dropped_bytes_count" AND resource.labels.zone = "us-central1-a"`, s)

	s = r.getFilterForMetric("pubsub.googleapis.com/subscription/ack_message_count")
	assert.Equal(t, `metric.type="pubsub.googleapis.com/subscription/ack_message_count"`, s)

	s = r.getFilterForMetric("loadbalancing.googleapis.com/https/backend_latencies")
	assert.Equal(t, `metric.type="loadbalancing.googleapis.com/https/backend_latencies"`, s)
}
