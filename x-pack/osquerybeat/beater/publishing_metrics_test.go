// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestPublishingMetricsNames(t *testing.T) {
	tests := []struct {
		queue string
		want  map[string]int64
	}{
		{
			queue: "scheduled",
			want: map[string]int64{
				"osquerybeat.publishing.scheduled.dropped.overflow":      1,
				"osquerybeat.publishing.scheduled.dropped_rows.overflow": 7,
				"osquerybeat.publishing.scheduled.dropped.expired":       0,
				"osquerybeat.publishing.scheduled.dropped_rows.expired":  0,
				"osquerybeat.publishing.scheduled.dropped.stale":         0,
				"osquerybeat.publishing.scheduled.dropped_rows.stale":    0,
				"osquerybeat.publishing.scheduled.dropped.shutdown":      0,
				"osquerybeat.publishing.scheduled.dropped_rows.shutdown": 0,
				"osquerybeat.publishing.scheduled.queue.items":           2,
				"osquerybeat.publishing.scheduled.queue.events":          9,
			},
		},
		{
			queue: "live",
			want: map[string]int64{
				"osquerybeat.publishing.live.dropped.stale":         0,
				"osquerybeat.publishing.live.dropped_rows.stale":    0,
				"osquerybeat.publishing.live.dropped.shutdown":      0,
				"osquerybeat.publishing.live.dropped_rows.shutdown": 0,
				"osquerybeat.publishing.live.dropped.timeout":       1,
				"osquerybeat.publishing.live.dropped_rows.timeout":  7,
				"osquerybeat.publishing.live.queue.items":           2,
				"osquerybeat.publishing.live.queue.events":          9,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.queue, func(t *testing.T) {
			reg := monitoring.NewRegistry()
			m := newPublishingMetrics(reg, test.queue, logptest.NewTestingLogger(t, ""))
			reason := dropOverflow
			if test.queue == "live" {
				reason = dropTimeout
			}
			m.drop(reason, 7)
			m.setQueue(2, 9)

			got := monitoring.CollectFlatSnapshot(reg, monitoring.Full, false).Ints
			assert.Equal(t, test.want, got, "the %s queue must report its drops in items and rows and its backlog", test.queue)
		})
	}
}

func TestPublishingMetricsWarnsOncePerMinute(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		log, logs := logptest.NewTestingLoggerWithObserver(t, "")
		m := newPublishingMetrics(monitoring.NewRegistry(), "scheduled", log)

		for range 100 {
			m.drop(dropOverflow, 1)
			time.Sleep(10 * time.Millisecond)
		}
		warned := logs.FilterMessageSnippet("osquerybeat.publishing.scheduled.dropped.overflow")
		assert.Equal(t, 1, warned.Len(), "100 drops in one second must log one warning naming the counter")
		assert.Equal(t, 1, logs.Len(), "drops must log nothing besides the warning")

		time.Sleep(time.Minute)
		m.drop(dropStale, 1)
		assert.Equal(t, 1, logs.FilterMessageSnippet("osquerybeat.publishing.scheduled.dropped.stale").Len(),
			"a drop a minute later must warn again, naming the counter that moved")

		time.Sleep(time.Minute)
		m.dropRows(dropShutdown, 1)
		assert.Equal(t, 1, logs.FilterMessageSnippet("osquerybeat.publishing.scheduled.dropped_rows.shutdown").Len(),
			"rows dropped without an item must name the rows counter")
		assert.Zero(t, logs.FilterMessageSnippet("dropped.shutdown").Len(), "the items counter did not move")
	})
}
