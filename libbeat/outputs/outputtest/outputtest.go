package outputtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type Metrics struct {
	Total      int64
	Acked      int64
	Dropped    int64
	Retryable  int64
	DeadLetter int64
	Duplicate  int64
	ErrTooMany int64

	Batches int64
}

func AssertOutputMetrics(t *testing.T, want Metrics, counter *beat.CountOutputListener, reg *monitoring.Registry) {
	t.Helper()

	metrics := monitoring.CollectStructSnapshot(reg, monitoring.Full, false)
	evs, ok := metrics["events"]
	require.True(t, ok, "could not find 'events' in metrics")
	parsedEvs, ok := evs.(map[string]any)
	require.True(t, ok, "could not parse 'events' isn't map[string]int64, it's %T", evs)

	// per-input metrics
	assert.Equal(t, want.Total, counter.NewLoad(), "per-input metric 'total'': unexpected value")
	assert.Equal(t, want.Acked, counter.AckedLoad(), "per-input metric 'acked': unexpected value")
	assert.Equal(t, want.Dropped, counter.DroppedLoad(), "per-input metric 'dropped': unexpected value")
	assert.Equal(t, want.Retryable, counter.RetryableErrorsLoad(), "per-input metric 'failed' (retryable error): unexpected value")

	assert.Equal(t, want.DeadLetter, counter.DeadLetterLoad(), "per-input metric 'dead_letter': unexpected value")
	assert.Equal(t, want.Duplicate, counter.DuplicateEventsLoad(), "per-input metric 'duplicates': unexpected value")
	assert.Equal(t, want.ErrTooMany, counter.ErrTooManyLoad(), "per-input metric 'toomany': unexpected value")

	// global output metrics
	assert.Equal(t, want.Total, parsedEvs["total"].(int64), "global metric 'total': unexpected value")
	assert.Equal(t, want.Acked, parsedEvs["acked"].(int64), "global metric 'acked': unexpected value")
	assert.Equal(t, want.Dropped, parsedEvs["dropped"].(int64), "global metric 'dropped': unexpected value")
	assert.Equal(t, want.Retryable, parsedEvs["failed"].(int64), "global metric 'failed' (retryable error): unexpected value")
	assert.Equal(t, want.Batches, parsedEvs["batches"].(int64), "global metric 'batches': unexpected value")

	assert.Equal(t, want.DeadLetter, parsedEvs["dead_letter"].(int64), "global metric 'dead_letter': unexpected value")
	assert.Equal(t, want.Duplicate, parsedEvs["duplicates"].(int64), "global metric 'duplicates': unexpected value")
	assert.Equal(t, want.ErrTooMany, parsedEvs["toomany"].(int64), "global metric 'toomany': unexpected value")
}
