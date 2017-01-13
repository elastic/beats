package module

import (
	"errors"

	"github.com/elastic/beats/libbeat/plugin"

	"github.com/elastic/beats/metricbeat/mb"
)

type modulePlugin struct {
	name       string
	factory    mb.ModuleFactory
	metricsets map[string]mb.MetricSetFactory
}

const pluginKey = "metricbeat.module"

func init() {
	plugin.MustRegisterLoader(pluginKey, func(ifc interface{}) error {
		p, ok := ifc.(modulePlugin)
		if !ok {
			return errors.New("plugin does not match metricbeat module plugin type")
		}

		if p.factory != nil {
			if err := mb.Registry.AddModule(p.name, p.factory); err != nil {
				return err
			}
		}

		for name, factory := range p.metricsets {
			if err := mb.Registry.AddMetricSet(p.name, name, factory); err != nil {
				return err
			}
		}

		return nil
	})
}

func Plugin(
	module string,
	factory mb.ModuleFactory,
	metricsets map[string]mb.MetricSetFactory,
) map[string][]interface{} {
	return plugin.MakePlugin(pluginKey, modulePlugin{module, factory, metricsets})
}

func MetricSetsPlugin(
	module string,
	metricsets map[string]mb.MetricSetFactory,
) map[string][]interface{} {
	return Plugin(module, nil, metricsets)
}
