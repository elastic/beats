// +build !integration

package logp

import (
	"expvar"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSnapshotExpvars(t *testing.T) {
	test := expvar.NewInt("test")
	test.Add(42)

	vals := map[string]int64{}
	snapshotExpvars(vals)

	assert.Equal(t, vals["test"], int64(42))
}

func TestSnapshotExpvarsMap(t *testing.T) {
	test := expvar.NewMap("testMap")
	test.Add("hello", 42)

	map2 := new(expvar.Map).Init()
	map2.Add("test", 5)
	test.Set("map2", map2)

	vals := map[string]int64{}
	snapshotExpvars(vals)

	assert.Equal(t, vals["testMap.hello"], int64(42))
	assert.Equal(t, vals["testMap.map2.test"], int64(5))
}

func TestBuildMetricsOutput(t *testing.T) {
	test := expvar.NewInt("testLog")
	test.Add(1)

	prevVals := map[string]int64{}
	snapshotExpvars(prevVals)

	test.Add(5)

	vals := map[string]int64{}
	snapshotExpvars(vals)

	metrics := buildMetricsOutput(prevVals, vals)
	assert.Equal(t, " testLog=5", metrics)
	prevVals = vals

	test.Add(3)
	vals = map[string]int64{}
	snapshotExpvars(vals)
	metrics = buildMetricsOutput(prevVals, vals)
	assert.Equal(t, " testLog=3", metrics)
}

func TestBuildMetricsOutputMissing(t *testing.T) {

	prevVals := map[string]int64{}
	snapshotExpvars(prevVals)

	test := expvar.NewInt("testLogEmpty")
	test.Add(7)

	vals := map[string]int64{}
	snapshotExpvars(vals)
	metrics := buildMetricsOutput(prevVals, vals)
	assert.Equal(t, " testLogEmpty=7", metrics)
}
