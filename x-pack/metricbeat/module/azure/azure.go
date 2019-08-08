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
	"github.com/pkg/errors"
	"time"
)

// Config options
type Config struct {
	ClientId            string           `config:"client_id"    validate:"required"`
	ClientSecret        string           `config:"client_secret" validate:"required"`
	TenantId            string           `config:"tenant_id" validate:"required"`
	SubscriptionId      string           `config:"subscription_id" validate:"required"`
	Period              time.Duration    `config:"period"`
	Resources           []ResourceConfig `config:"resources"`
	RefreshListInterval time.Duration    `config:"refresh_list_interval"`
}

// MetricConfig contains metric specific configuration.
type ResourceConfig struct {
	Id      string         `config:"resource_id"`
	Group   string         `config:"resource_group"`
	Metrics []MetricConfig `config:"metrics"`
	Type    string         `config:"resource_type"`
	Query   string         `config:"resource_query"`
}

type MetricConfig struct {
	Name         []string          `config:"name"`
	Namespace    string            `config:"namespace"`
	Aggregations []string          `config:"aggregations"`
	Dimensions   []DimensionConfig `config:"dimensions"`
}

type DimensionConfig struct {
	Name  string `config:"name"`
	Value string `config:"value"`
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
		return nil, errors.Wrap(err, "error unpack raw module config using UnpackConfig")
	}
	return &base, nil
}
