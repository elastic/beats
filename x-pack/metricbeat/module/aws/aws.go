package aws

import "github.com/elastic/beats/metricbeat/mb"

// Config defines all required and optional parameters for aws metricsets
type Config struct {
	Mock            string `config:"mock" validate:"nonzero,required"`
	AccessKeyID     string `config:"access_key_id" validate:"nonzero"`
	SecretAccessKey string `config:"secret_access_key" validate:"nonzero"`
	SessionToken    string `config:"session_token" validate:"nonzero"`
}

// MetricSet is the base metricset for all aws metricsets
type MetricSet struct {
	mb.BaseMetricSet
}

func init() {
	if err := mb.Registry.AddModule("aws", newModule); err != nil {
		panic(err)
	}
}

func newModule(base mb.BaseModule) (mb.Module, error) {
	var config Config
	if err := base.UnpackConfig(&config); err != nil {
		return nil, err
	}
	return &base, nil
}

// NewMetricSet creates a base metricset for aws metricsets
func NewMetricSet(base mb.BaseMetricSet) (*MetricSet, error) {
	var config Config
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}
	return &MetricSet{BaseMetricSet: base}, nil
}
