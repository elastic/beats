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
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/include"
)

type Metricbeat struct {
	done   chan struct{}
	config *Config
}

// New creates and returns a new Metricbeat instance.
func New() *Metricbeat {
	return &Metricbeat{}
}

func (mb *Metricbeat) Config(b *beat.Beat) error {
	mb.config = &Config{}
	err := b.RawConfig.Unpack(mb.config)
	if err != nil {
		return fmt.Errorf("error reading configuration file. %v", err)
	}

	// List all registered modules and metricsets
	include.ListAll()

	return nil
}

func (mb *Metricbeat) Setup(b *beat.Beat) error {
	mb.done = make(chan struct{})
	return nil
}

func (mb *Metricbeat) Run(b *beat.Beat) error {
	// Checks all defined metricsets and starts a module for each entry with the defined metricsets
	for _, moduleConfig := range mb.config.Metricbeat.Modules {

		module, err := helper.Registry.GetModule(moduleConfig)
		if err != nil {
			return err
		}

		err = module.Start(b)
		if err != nil {
			return err
		}
	}

	<-mb.done

	return nil
}

func (mb *Metricbeat) Cleanup(b *beat.Beat) error {
	return nil
}

func (mb *Metricbeat) Stop() {
	close(mb.done)
}
