package elasticsearch

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/stretchr/testify/assert"
)

func TestSnapshot(t *testing.T) {
	tests := []struct {
		name     string
		expected common.MapStr
		build    func(R *monitoring.Registry)
	}{
		{
			"empty registry",
			nil,
			func(*monitoring.Registry) {},
		},
		{
			"empty if metric is not exposed",
			nil,
			func(R *monitoring.Registry) {
				monitoring.NewInt(R, "test").Set(1)
			},
		},
		{
			"collect exposed metric",
			common.MapStr{"test": int64(1)},
			func(R *monitoring.Registry) {
				monitoring.NewInt(R, "test", monitoring.Report).Set(1)
			},
		},
		{
			"do not report unexported namespace",
			common.MapStr{"test": int64(0)},
			func(R *monitoring.Registry) {
				monitoring.NewInt(R, "test", monitoring.Report)
				monitoring.NewInt(R, "unexported.test")
			},
		},
		{
			"do not report empty nested exported",
			common.MapStr{"test": int64(0)},
			func(R *monitoring.Registry) {
				metrics := R.NewRegistry("exported", monitoring.Report)
				monitoring.NewInt(metrics, "unexported", monitoring.DoNotReport)
				monitoring.NewInt(R, "test", monitoring.Report)
			},
		},
		{
			"export namespaced as nested-document from registry instance",
			common.MapStr{"exported": common.MapStr{"test": int64(0)}},
			func(R *monitoring.Registry) {
				metrics := R.NewRegistry("exported", monitoring.Report)
				monitoring.NewInt(metrics, "test", monitoring.Report)
				monitoring.NewInt(R, "unexported.test")
			},
		},
		{
			"export unmarked namespaced as nested-document from registry instance",
			common.MapStr{"exported": common.MapStr{"test": int64(0)}},
			func(R *monitoring.Registry) {
				metrics := R.NewRegistry("exported", monitoring.Report)
				monitoring.NewInt(metrics, "test")
				monitoring.NewInt(R, "unexported.test")
			},
		},
		{
			"export namespaced as nested-document without intermediate registry instance",
			common.MapStr{"exported": common.MapStr{"test": int64(0)}},
			func(R *monitoring.Registry) {
				monitoring.NewInt(R, "exported.test", monitoring.Report)
				monitoring.NewInt(R, "unexported.test")
			},
		},
	}

	for i, test := range tests {
		t.Logf("run test (%v - %v): %v", i, test.name, test.expected)

		R := monitoring.NewRegistry()
		test.build(R)
		snapshot := makeSnapshot(R)

		t.Logf("  actual: %v", snapshot)
		assert.Equal(t, test.expected, snapshot)
	}
}
