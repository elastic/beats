// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package health

import (
	"encoding/json"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
	s "github.com/elastic/beats/v8/libbeat/common/schema"
	c "github.com/elastic/beats/v8/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v8/metricbeat/helper/elastic"
	"github.com/elastic/beats/v8/metricbeat/mb"
)

var (
	schema = s.Schema{
		"name": c.Str("name"),

		"version": c.Dict("version", s.Schema{
			"number":     c.Str("number"),
			"build_hash": c.Str("build_hash"),
		}),

		"jvm": c.Dict("jvm", s.Schema{
			"version": c.Str("version"),

			"gc": c.Dict("gc", s.Schema{
				"collection_count": c.Int("collection_count"),
				"collection_time":  s.Object{"ms": c.Int("collection_time")},
				// TODO: Add separate metrics for old and young generation collectors
			}),

			"memory_usage": c.Dict("memory_usage", s.Schema{
				"heap_init":                         s.Object{"bytes": c.Int("heap_init")},
				"heap_used":                         s.Object{"bytes": c.Int("heap_used")},
				"heap_committed":                    s.Object{"bytes": c.Int("heap_committed")},
				"heap_max":                          s.Object{"bytes": c.Int("heap_max")},
				"non_heap_init":                     s.Object{"bytes": c.Int("non_heap_init")},
				"non_heap_committed":                s.Object{"bytes": c.Int("non_heap_committed")},
				"object_pending_finalization_count": c.Int("object_pending_finalization_count"),
			}),

			"threads": c.Dict("threads", s.Schema{
				"current":       c.Int("thread_count"),
				"daemon":        c.Int("daemon_thread_count"),
				"max":           c.Int("peak_thread_count"),
				"total_started": c.Int("total_started_thread_count"),
			}),
		}),

		"process": c.Dict("process", s.Schema{
			"pid":    c.Int("pid"),
			"uptime": s.Object{"sec": c.Int("uptime")},

			"filebeat": c.Dict("filebeat", s.Schema{
				"pid":                     c.Int("pid"),
				"restart_count":           c.Int("restart_count"),
				"time_since_last_restart": s.Object{"sec": c.Int("seconds_since_last_restart")},
			}),
		}),

		"crawler": c.Dict("crawler", s.Schema{
			"workers": c.Dict("workers", s.Schema{
				"pool_size": c.Int("pool_size"),
				"active":    c.Int("active"),
				"available": c.Int("available"),
			}),
		}),
	}
)

func eventMapping(report mb.ReporterV2, input []byte, isXpack bool) error {
	var data map[string]interface{}
	err := json.Unmarshal(input, &data)
	if err != nil {
		return err
	}
	var errs multierror.Errors

	// All events need to have a cluster_uuid to work with Stack Monitoring
	event := mb.Event{
		ModuleFields:    common.MapStr{},
		MetricSetFields: common.MapStr{},
	}
	event.ModuleFields.Put("cluster_uuid", data["cluster_uuid"])

	// Collect process info in a form ready for mapping
	process := make(map[string]interface{})
	process["filebeat"] = data["filebeat"]

	jvm, ok := data["jvm"].(map[string]interface{})
	if ok {
		if pid, ok := jvm["pid"]; ok {
			process["pid"] = pid
		}
		if uptime, ok := jvm["uptime"]; ok {
			process["uptime"] = uptime
		}

		// Add version info to the JVM section to help the schema mapper find it
		system, ok := data["system"].(map[string]interface{})
		if ok {
			jvm["version"] = system["java_version"]
		} else {
			errs = append(errs, errors.New("system is not a map"))
		}
	} else {
		errs = append(errs, errors.New("jvm is not a map"))
	}

	// Set the process info we have collected
	data["process"] = process

	// xpack.enabled in config using standalone metricbeat writes to `.monitoring` instead of `metricbeat-*`
	// When using Agent, the index name is overwritten anyways.
	if isXpack {
		index := elastic.MakeXPackMonitoringIndexName(elastic.EnterpriseSearch)
		event.Index = index
	}

	event.MetricSetFields, err = schema.Apply(data)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "failure to apply health schema"))
	} else {
		report.Event(event)
	}

	return errs.Err()
}
