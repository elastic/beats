/*
Package include imports all Module and MetricSet packages so that they register
their factories with the global registry. This package can be imported in the
main package to automatically register all of the standard supported Metricbeat
modules.
*/
package include

import (
	// Every module and metricset must be added here so that they can register
	// themselves.
	_ "github.com/elastic/beats/metricbeat/module/apache"
	_ "github.com/elastic/beats/metricbeat/module/apache/status"
	_ "github.com/elastic/beats/metricbeat/module/mysql"
	_ "github.com/elastic/beats/metricbeat/module/mysql/status"
	_ "github.com/elastic/beats/metricbeat/module/redis"
	_ "github.com/elastic/beats/metricbeat/module/redis/info"
	_ "github.com/elastic/beats/metricbeat/module/system"
	_ "github.com/elastic/beats/metricbeat/module/system/cores"
	_ "github.com/elastic/beats/metricbeat/module/system/cpu"
	_ "github.com/elastic/beats/metricbeat/module/system/filesystem"
	_ "github.com/elastic/beats/metricbeat/module/system/fsstats"
	_ "github.com/elastic/beats/metricbeat/module/system/memory"
	_ "github.com/elastic/beats/metricbeat/module/system/process"
	_ "github.com/elastic/beats/metricbeat/module/zookeeper"
	_ "github.com/elastic/beats/metricbeat/module/zookeeper/mntr"
)
