// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

/*
Package mysql is Metricbeat module for MySQL server.
*/

package azure

import (
	"github.com/elastic/beats/metricbeat/mb"
)

// Config options
type Config struct {
	ClientId       string `config:"client_id"    validate:"required"`
	ClientSecret   string `config:"client_secret"`
	TenantId       string `config:"tenant_id" validate:"required"`
	SubscriptionId string `config:"subscription_id" validate:"required"`
}

func init() {
	// Register the ModuleFactory function for the "azure" module.
	if err := mb.Registry.AddModule("azure", newModule); err != nil {
		panic(err)
	}
}

// newModule adds validation that hosts is non-empty, a requirement to use the
// azure module.
func newModule(base mb.BaseModule) (mb.Module, error) {
	var config Config
	if err := base.UnpackConfig(&config); err != nil {
		return nil, err
	}
	return &base, nil
}

// NewMetricSet creates a base metricset for default configurations optons and auth in the future
func GetConfig(base mb.BaseMetricSet) (Config, error) {
	var config Config
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return config, err
	}
	return config, nil
}
