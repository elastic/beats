// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import "github.com/elastic/beats/metricbeat/mb"

// Config defines all required and optional parameters for aws metricsets
type Config struct {
	Period          string `config:"period"`
	AccessKeyID     string `config:"access_key_id"`
	SecretAccessKey string `config:"secret_access_key"`
	SessionToken    string `config:"session_token"`
	DefaultRegion   string `config:"default_region"`
}

// MetricSet is the base metricset for all aws metricsets
type MetricSet struct {
	mb.BaseMetricSet
}

// ModuleName is the name of this module.
const ModuleName = "aws"

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

// NewMetricSet creates a base metricset for aws metricsets
func NewMetricSet(base mb.BaseMetricSet) (*MetricSet, error) {
	var config Config
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}
	return &MetricSet{BaseMetricSet: base}, nil
}
