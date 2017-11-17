package monitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSnapshot(t *testing.T) {
	tests := []struct {
		name     string
		expected map[string]interface{}
		build    func(R *Registry)
	}{
		{
			"empty registry",
			nil,
			func(*Registry) {},
		},
		{
			"empty if metric is not exposed",
			nil,
			func(R *Registry) {
				NewInt(R, "test").Set(1)
			},
		},
		{
			"collect exposed metric",
			map[string]interface{}{"test": int64(1)},
			func(R *Registry) {
				NewInt(R, "test", Report).Set(1)
			},
		},
		{
			"do not report unexported namespace",
			map[string]interface{}{"test": int64(0)},
			func(R *Registry) {
				NewInt(R, "test", Report)
				NewInt(R, "unexported.test")
			},
		},
		{
			"do not report empty nested exported",
			map[string]interface{}{"test": int64(0)},
			func(R *Registry) {
				metrics := R.NewRegistry("exported", Report)
				NewInt(metrics, "unexported", DoNotReport)
				NewInt(R, "test", Report)
			},
		},
		{
			"export namespaced as nested-document from registry instance",
			map[string]interface{}{"exported": map[string]interface{}{"test": int64(0)}},
			func(R *Registry) {
				metrics := R.NewRegistry("exported", Report)
				NewInt(metrics, "test", Report)
				NewInt(R, "unexported.test")
			},
		},
		{
			"export unmarked namespaced as nested-document from registry instance",
			map[string]interface{}{"exported": map[string]interface{}{"test": int64(0)}},
			func(R *Registry) {
				metrics := R.NewRegistry("exported", Report)
				NewInt(metrics, "test")
				NewInt(R, "unexported.test")
			},
		},
		{
			"export namespaced as nested-document without intermediate registry instance",
			map[string]interface{}{"exported": map[string]interface{}{"test": int64(0)}},
			func(R *Registry) {
				NewInt(R, "exported.test", Report)
				NewInt(R, "unexported.test")
			},
		},
	}

	for i, test := range tests {
		t.Logf("run test (%v - %v): %v", i, test.name, test.expected)

		R := NewRegistry()
		test.build(R)
		snapshot := CollectStructSnapshot(R, Reported, false)

		t.Logf("  actual: %v", snapshot)
		assert.Equal(t, test.expected, snapshot)
	}
}
