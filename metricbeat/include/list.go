/*

This file is included in the main file to load all metricsets.

In case only a subset of metricsets should be included, they can be specified manually in the main.go file.

*/
package include

// Make sure all active plugins are loaded
// TODO: create a script to automatically generate this list
import (
	"github.com/elastic/beats/libbeat/logp"
	_ "github.com/elastic/beats/metricbeat/helper"

	// List of all metrics to make sure they are registred
	// Every new metric must be added here
	_ "github.com/elastic/beats/metricbeat/module/apache/status"
	_ "github.com/elastic/beats/metricbeat/module/golang/expvar"
	_ "github.com/elastic/beats/metricbeat/module/mysql/status"
	_ "github.com/elastic/beats/metricbeat/module/redis/info"
)

func ListAll() {
	logp.Debug("beat", "Registered Modules and Metrics")
	//for moduleName, module := range helper.Registry {
	//	for metricName, _ := range module.MetricSets {
	//		logp.Debug("beat", "Registred: Module: %v, Metric: %v", moduleName, metricName)
	//	}
	//}
}
