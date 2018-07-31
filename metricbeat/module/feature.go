package module

import (
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/metricbeat/mb"
)

// MetricSetFeature creates a new MetricSet feature.
func MetricSetFeature(
	module, name string,
	ms *mb.MetricSetRegistration,
	description feature.Describer,
) *feature.Feature {
	return feature.New(namespace(module), name, ms, description)
}

// Feature creates a new Module feature.
func Feature(
	module string,
	factory mb.ModuleFactory,
	description feature.Describer,
) *feature.Feature {
	return feature.New(mb.Namespace, module, factory, description)
}

func namespace(module string) string {
	return mb.Namespace + "." + module
}
