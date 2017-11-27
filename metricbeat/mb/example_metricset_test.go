package mb_test

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

var hostParser = parse.URLHostParserBuilder{
	DefaultScheme: "http",
}.Build()

func init() {
	// Register the MetricSetFactory function for the "status" MetricSet.
	mb.Registry.MustAddMetricSet("someapp", "status", NewMetricSet,
		mb.WithHostParser(hostParser),
	)
}

type MetricSet struct {
	mb.BaseMetricSet
}

func NewMetricSet(base mb.BaseMetricSet) (mb.MetricSet, error) {
	fmt.Println("someapp-status url=", base.HostData().SanitizedURI)
	return &MetricSet{BaseMetricSet: base}, nil
}

// Fetch will be called periodically by the framework.
func (ms *MetricSet) Fetch(report mb.Reporter) {
	// Fetch data from the host at ms.HostData().URI and return the data.
	data, err := common.MapStr{
		"some_metric":          18.0,
		"answer_to_everything": 42,
	}, error(nil)
	if err != nil {
		// Report an error if it occurs.
		report.Error(err)
		return
	}

	// Otherwise report the collected data.
	report.Event(data)
}

// ExampleReportingMetricSet demonstrates how to register a MetricSetFactory
// and implement a ReportingMetricSet.
func ExampleReportingMetricSet() {}
