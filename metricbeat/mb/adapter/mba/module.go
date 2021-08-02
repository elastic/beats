package mba

import "github.com/elastic/beats/v7/metricbeat/mb"

// ModuleAdapter creates a shared module instance for creating custom inputs based on
// metricsets.
type ModuleAdapter struct {
	Name      string
	Factory   mb.ModuleFactory
	Modifiers []mb.EventModifier
}

func (ma *ModuleAdapter) ModuleName() string                 { return ma.Name }
func (ma *ModuleAdapter) EventModifiers() []mb.EventModifier { return ma.Modifiers }
func (ma *ModuleAdapter) Create(base mb.BaseModule) (mb.Module, error) {
	if ma.Factory != nil {
		return ma.Factory(base)
	}
	return mb.DefaultModuleFactory(base)
}

func (ma *ModuleAdapter) MetricsetInput(inputName string, metricsetName string, factory mb.MetricSetFactory) MetricsetManager {
	return MetricsetManager{
		InputName:     inputName,
		MetricsetName: metricsetName,
		ModuleManager: ma,
		Factory:       factory,
	}
}
