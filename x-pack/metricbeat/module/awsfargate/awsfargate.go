// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awsfargate

import (
	"time"

	"github.com/menderesk/beats/v7/metricbeat/mb"
)

// Config defines all required and optional parameters for awsfargate metricsets
type Config struct {
	Period time.Duration `config:"period" validate:"nonzero,required"`
}

// MetricSet is the base metricset for all aws metricsets
type MetricSet struct {
	mb.BaseMetricSet
	Period time.Duration
}

// ModuleName is the name of this module.
const ModuleName = "awsfargate"

func init() {
	if err := mb.Registry.AddModule(ModuleName, newModule); err != nil {
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

// NewMetricSet creates a base metricset for awsfargate metricset
func NewMetricSet(base mb.BaseMetricSet) (*MetricSet, error) {
	var config Config
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	metricSet := MetricSet{
		BaseMetricSet: base,
		Period:        config.Period,
	}
	return &metricSet, nil
}
