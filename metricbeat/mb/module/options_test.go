package module

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/metricbeat/mb"
)

func TestWithMaxStartDelay(t *testing.T) {
	w := &Wrapper{}
	WithMaxStartDelay(1)(w)
	assert.EqualValues(t, 1, w.maxStartDelay)
}

func TestWithMetricSetInfo(t *testing.T) {
	w := &Wrapper{}
	WithMetricSetInfo()(w)
	assert.Len(t, w.eventModifiers, 1)
}

func TestWithEventModifier(t *testing.T) {
	f1 := func(module, metricset string, event *mb.Event) {}
	f2 := func(module, metricset string, event *mb.Event) {}

	w := &Wrapper{}
	WithEventModifier(f1)(w)
	WithEventModifier(f2)(w)
	assert.Len(t, w.eventModifiers, 2)
}
