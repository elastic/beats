// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package system

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/go-sysinfo"
)

func init() {
	// Register the custom ModuleFactory function for the system module.
	if err := mb.Registry.AddModule("system", NewModule); err != nil {
		panic(err)
	}
}

// SystemModuleConfig contains the configuration specific to the system module.
type SystemModuleConfig struct {
	// In Auditbeat, sub-modules are called datasets. This extends the module
	// configuration to allow specifying them under "datasets" rather than
	// "metricsets".
	DataSets []string `config:"datasets"`
}

// SystemModule extends the Metricbeat BaseModule. The purpose is to allow
// configuring sub-modules as "datasets" rather than "metricsets".
type SystemModule struct {
	mb.BaseModule
	config SystemModuleConfig
	hostID string
}

// Config returns the ModuleConfig used to create the Module.
// It overwrites MetricSets with the configured datasets.
func (m *SystemModule) Config() mb.ModuleConfig {
	config := m.BaseModule.Config()
	config.MetricSets = m.config.DataSets
	return config
}

// NewModule creates a new mb.Module instance.
func NewModule(base mb.BaseModule) (mb.Module, error) {
	var config SystemModuleConfig
	if err := base.UnpackConfig(&config); err != nil {
		return nil, err
	}

	hostInfo, err := sysinfo.Host()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get host ID")
	}

	return &SystemModule{
		BaseModule: base,
		config:     config,
		hostID:     hostInfo.Info().UniqueID,
	}, nil
}

// SystemMetricSet extends the Metricbeat BaseMetricSet.
type SystemMetricSet struct {
	mb.BaseMetricSet
	module *SystemModule
}

// NewSystemMetricSet creates a new SystemMetricSet.
func NewSystemMetricSet(base mb.BaseMetricSet) SystemMetricSet {
	return SystemMetricSet{
		BaseMetricSet: base,
		module:        base.Module().(*SystemModule),
	}
}

// HostID returns the stored host ID.
func (ms *SystemMetricSet) HostID() string {
	return ms.module.hostID
}
