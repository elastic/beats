package mb_test

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

func init() {
	// Register the MetricSetFactory function for the "status" MetricSet.
	if err := mb.Registry.AddMetricSet("someapp", "status", NewMetricSet); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	username string
	password string
}

func NewMetricSet(base mb.BaseMetricSet) (mb.MetricSet, error) {
	// Unpack additional configuration options.
	config := struct {
		Username string `config:"username"`
		Password string `config:"password"`
	}{
		Username: "",
		Password: "",
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		username:      config.Username,
		password:      config.Password,
	}, nil
}

func (ms *MetricSet) Fetch() (common.MapStr, error) {
	// Fetch data from host and return the data.
	return common.MapStr{
		"someParam":  "value",
		"otherParam": 42,
	}, nil
}

// ExampleMetricSetFactory demonstrates how to register a MetricSetFactory
// and unpack additional configuration data.
func ExampleMetricSetFactory() {}
