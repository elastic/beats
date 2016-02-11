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
	"github.com/urso/ucfg"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper"
)

type Metricbeat struct {
	done          chan struct{}
	Configuration *MetricbeatConfig
}

// New creates a new Metricbeat instance
func New() *Metricbeat {
	return &Metricbeat{}
}

func readMetricbeatConfig() (*MetricbeatConfig, error) {
	// TODO: replace reading + parsing via ucfg.yaml.NewConfigWithFile
	var rawYAMLConfig map[string]interface{}
	err := cfgfile.Read(&rawYAMLConfig, "")
	if err != nil {
		logp.Err("Error reading configuration file: %v", err)
		return nil, err
	}

	rawConfig := ucfg.New()
	err = rawConfig.Merge(map[string]interface{}{
		"metricbeat": rawYAMLConfig["metricbeat"],
	})
	if err != nil {
		logp.Err("Error reading configuration file: %v", err)
		return nil, err
	}

	config := &MetricbeatConfig{}
	err = rawConfig.Unpack(config)
	if err != nil {
		logp.Err("Error reading configuration file: %v", err)
		return nil, err
	}

	return config, nil
}

func (mb *Metricbeat) Config(b *beat.Beat) error {
	config, err := readMetricbeatConfig()
	if err != nil {
		return err
	}

	mb.Configuration = config

	logp.Info("Setup base and raw configuration for Modules and Metrics")

	for moduleName, moduleCfg := range config.Metricbeat.Modules {
		module, ok := helper.Registry[moduleName]
		if !ok {
			logp.Critical("Unknown module: %v", moduleName)
			continue
		}

		module.Enabled = true
		module.Config = moduleCfg

		var metricsConfig struct {
			MetricSets map[string]*ucfg.Config
		}
		moduleCfg.Unpack(&metricsConfig)

		for metric, metricCfg := range metricsConfig.MetricSets {
			metricSet, ok := module.MetricSets[metric]
			if !ok {
				logp.Critical("Unknown module metric: %v.%v", moduleName, metric)
				continue
			}

			metricSet.Enabled = true
			metricSet.Config = metricCfg
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
