package mb_test

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

var hostParser = parse.URLHostParserBuilder{DefaultScheme: "http"}.Build()

func init() {
	// Register the MetricSetFactory function for the "status" MetricSet.
	if err := mb.Registry.AddMetricSet("someapp", "status", NewMetricSet, hostParser); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
}

func NewMetricSet(base mb.BaseMetricSet) (mb.MetricSet, error) {
	fmt.Println("someapp-status url=", base.HostData().SanitizedURI)
	return &MetricSet{BaseMetricSet: base}, nil
}

func (ms *MetricSet) Fetch() (common.MapStr, error) {
	// Fetch data from the host (using ms.HostData().URI) and return the data.
	return common.MapStr{
		"someParam":  "value",
		"otherParam": 42,
	}, nil
}

// ExampleMetricSetFactory demonstrates how to register a MetricSetFactory
// and unpack additional configuration data.
func ExampleMetricSetFactory() {}
