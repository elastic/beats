// Package mba provides adapters for using metricbeat modules and metricsets as v2 inputs.
//
// The metricsets provided by these wrappers are mostly independent of the
// Metricbeat core framework. No globals or shared state in the metricbeat core
// will be read or written. Each MeticsetManager will be fully independent.
//
// The MetricsetManager is used to wrap a Metricbeat module and its Metricsets.
//
// In Metricbeat a module implementation is optional. The module
// implementations purpose is to provide additional functionality (parsing,
// query, cache data, coordination/sharing) for its metricsets. A module
// instance will be shared between all metricsets.
// Independent if a Module is implemnted or not, an mb.Module instance will be
// passed to the metricset.
//
// The adapters provided in this package provide similar functionality to metricbeat. The sharing
// of Modules with metricsets requires authors to create a common ModuleAdapter, that will be shared between
// add metricset input adapters.
//
// Note: The ModuleAdapter should also be used for metricsets that do not require a shared module.
//
//
// Example system module:
//
// ```
//
// func systemMetricsPlugins() []v2.Plugin {
//	// Create shared module adapter. The Factory is optional. The adapter
//	// provides helpers to create inputs from metricsets.
//	systemModule := &mba.ModuleAdapter{Name: "system", Factory: system.NewModule}
//
//
//	// Create list of inputs from metricset implementations:
//	return []v2.Plugin{
//			systemModule.MetricsetInput("system.cpu", "cpu", cpu.New),
//			...
//	}
// }
//
// ```
//
// Metricbeat allows developers to pass additional options when registering with the mb.Registry.
// The optionas can provide simple meta-data, some form of config validation, or additional hooks to modify some of
// the default behavior (e.g. HostParser). The MetricsetManager returned by
// (*ModuleAdapter).MetricsetInput can be modified directly, or using one of its WithX methods.
package mba

//go:generate godocdown -plain=false -output Readme.md

import (
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/feature"
)

// Plugin create a v2.Plugin for a MetricsetManager.
func Plugin(stability feature.Stability, deprecated bool, mm MetricsetManager) v2.Plugin {
	return v2.Plugin{
		Name:       mm.InputName,
		Stability:  stability,
		Deprecated: deprecated,
		Manager:    &mm,
	}
}
