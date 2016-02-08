/*

Metricbeat collects metric sets from different modules.


Each event created has the following format:

	curl -XPUT http://localhost:9200/metricbeat/metricsets -d
	{
		"metriset": metricsetName,
		"module": moduleName,
		"moduleName-metricSetName": {
			"metric1": "value",
			"metric2": "value",
			"metric3": "value",
			"nestedmetric": {
				"metric4": "value"
			}
		},
		"@timestamp": timestamp
	}

All documents are currently stored in one index called metricbeat. It is important to use an independent namespace
for each MetricSet to prevent type conflicts. Also all values are stored under the same type "metricsets".

*/
package beater

import (
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper"
)

type Metricbeat struct {
	done          chan struct{}
	MbConfig      *MetricbeatConfig
	ModulesConfig *RawModulesConfig
	MetricsConfig *RawMetricsConfig
}

// New creates a new Metricbeat instance
func New() *Metricbeat {
	return &Metricbeat{}
}

func (mb *Metricbeat) Config(b *beat.Beat) error {

	mb.MbConfig = &MetricbeatConfig{}
	err := cfgfile.Read(mb.MbConfig, "")
	if err != nil {
		fmt.Println(err)
		logp.Err("Error reading configuration file: %v", err)
		return err
	}

	mb.ModulesConfig = &RawModulesConfig{}
	err = cfgfile.Read(mb.ModulesConfig, "")
	if err != nil {
		fmt.Println(err)
		logp.Err("Error reading configuration file: %v", err)
		return err
	}

	mb.MetricsConfig = &RawMetricsConfig{}
	err = cfgfile.Read(mb.MetricsConfig, "")
	if err != nil {
		fmt.Println(err)
		logp.Err("Error reading configuration file: %v", err)
		return err
	}

	logp.Info("Setup base and raw configuration for Modules and Metrics")
	// Apply the base configuration to each module and metric
	for moduleName, module := range helper.Registry {
		// Check if config for module exist. Only configured modules are loaded
		if _, ok := mb.MbConfig.Metricbeat.Modules[moduleName]; !ok {
			continue
		}
		module.BaseConfig = mb.MbConfig.getModuleConfig(moduleName)
		module.RawConfig = mb.ModulesConfig.Metricbeat.Modules[moduleName]
		module.Enabled = true

		for metricSetName, metricSet := range module.MetricSets {
			// Check if config for metricset exist. Only configured metricset are loaded
			if _, ok := mb.MbConfig.getModuleConfig(moduleName).MetricSets[metricSetName]; !ok {
				continue
			}
			metricSet.BaseConfig = mb.MbConfig.getModuleConfig(moduleName).MetricSets[metricSetName]
			metricSet.RawConfig = mb.MetricsConfig.Metricbeat.Modules[moduleName].MetricSets[metricSetName]
			metricSet.Enabled = true
		}
	}

	return nil
}

func (mb *Metricbeat) Setup(b *beat.Beat) error {
	mb.done = make(chan struct{})
	return nil
}

func (mb *Metricbeat) Run(b *beat.Beat) error {

	helper.StartModules(b)
	<-mb.done

	return nil
}

func (mb *Metricbeat) Cleanup(b *beat.Beat) error {
	return nil
}

func (mb *Metricbeat) Stop() {
	close(mb.done)
}
