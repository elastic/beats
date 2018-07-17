// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package stats

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	schema = s.Schema{
		"cluster_uuid":           c.Str("cluster_uuid"),
		"concurrent_connections": c.Int("concurrent_connections"),
		"os": c.Dict("os", s.Schema{
			"load": c.Dict("load", s.Schema{
				"avg": s.Object{
					"1m":  c.Float("1m"),
					"5m":  c.Float("5m"),
					"15m": c.Float("15m"),
				},
			}),
			"memory": c.Dict("memory", s.Schema{
				"total": s.Object{
					"bytes": c.Int("total_bytes"),
				},
				"free": s.Object{
					"bytes": c.Int("free_bytes"),
				},
				"used": s.Object{
					"bytes": c.Int("used_bytes"),
				},
			}),
			"uptime": s.Object{
				"ms": c.Int("uptime_ms"),
			},
		}),
		"process": c.Dict("process", s.Schema{
			"event_loop_delay": s.Object{
				"ms": c.Float("event_loop_delay"),
			},
			"memory": c.Dict("memory", s.Schema{
				"heap": c.Dict("heap", s.Schema{
					"total": s.Object{
						"bytes": c.Int("total_bytes"),
					},
					"used": s.Object{
						"bytes": c.Int("used_bytes"),
					},
					"size_limit": s.Object{
						"bytes": c.Int("size_limit"),
					},
				}),
			}),
			"uptime": s.Object{
				"ms": c.Int("uptime_ms"),
			},
		}),
		"requests": RequestsDict,
		"response_times": c.Dict("response_times", s.Schema{
			"avg": s.Object{
				"ms": c.Float("avg_ms"),
			},
			"max": s.Object{
				"ms": c.Float("max_ms"),
			},
		}),
		"kibana": KibanaDict,
		"usage": c.Dict("usage", s.Schema{
			"kibana":    kibanaUsageDict,
			"reporting": ReportingUsageDict,
		}),
	}

	// RequestsDict defines how to convert the requests field
	RequestsDict = c.Dict("requests", s.Schema{
		"disconnects": c.Int("disconnects", s.Optional),
		"total":       c.Int("total", s.Optional),
	})

	// KibanaDict defines how to convert the kibana field
	KibanaDict = c.Dict("kibana", s.Schema{
		"uuid":              c.Str("uuid"),
		"name":              c.Str("name"),
		"index":             c.Str("index"),
		"host":              c.Str("host"),
		"transport_address": c.Str("transport_address"),
		"version":           c.Str("version"),
		"snapshot":          c.Bool("snapshot"),
		"status":            c.Str("status"),
	})

	kibanaUsageDict = c.Dict("kibana", s.Schema{
		"index": c.Str("index"),
		"dashboard": c.Dict("dashboard", s.Schema{
			"total": c.Int("total"),
		}, c.DictOptional),
		"visualization": c.Dict("visualization", s.Schema{
			"total": c.Int("total"),
		}, c.DictOptional),
		"search": c.Dict("search", s.Schema{
			"total": c.Int("total"),
		}, c.DictOptional),
		"index_pattern": c.Dict("index_pattern", s.Schema{
			"total": c.Int("total"),
		}, c.DictOptional),
		"graph_workspace": c.Dict("graph_workspace", s.Schema{
			"total": c.Int("total"),
		}, c.DictOptional),
		"timelion_sheet": c.Dict("timelion_sheet", s.Schema{
			"total": c.Int("total"),
		}, c.DictOptional),
	})

	// ReportingUsageDict defines how to convert the usage.reporting field
	ReportingUsageDict = c.Dict("reporting", s.Schema{
		"available":     c.Bool("available"),
		"enabled":       c.Bool("enabled"),
		"browser_type":  c.Str("browser_type"),
		"all":           c.Int("all"),
		"csv":           reportingCsvDict,
		"printable_pdf": reportingPrintablePdfDict,
		"status":        reportingStatusDict,
		"lastDay":       c.Dict("lastDay", reportingPeriodSchema, c.DictOptional),
		"last7Days":     c.Dict("last7Days", reportingPeriodSchema, c.DictOptional),
	}, c.DictOptional)

	reportingCsvDict = c.Dict("csv", s.Schema{
		"available": c.Bool("available"),
		"total":     c.Int("total"),
	}, c.DictOptional)

	reportingPrintablePdfDict = c.Dict("printable_pdf", s.Schema{
		"available": c.Bool("available"),
		"total":     c.Int("total"),
	}, c.DictOptional)

	reportingStatusDict = c.Dict("status", s.Schema{
		"completed":  c.Int("completed", s.Optional),
		"failed":     c.Int("failed", s.Optional),
		"processing": c.Int("processing", s.Optional),
		"pending":    c.Int("pending", s.Optional),
	}, c.DictOptional)

	reportingPeriodSchema = s.Schema{
		"all":           c.Int("all"),
		"csv":           reportingCsvDict,
		"printable_pdf": reportingPrintablePdfDict,
		"status":        reportingStatusDict,
	}
)

func eventMapping(r mb.ReporterV2, content []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		r.Error(err)
		return err
	}

	dataFields, err := schema.Apply(data)
	if err != nil {
		r.Error(err)
	}

	var event mb.Event
	event.RootFields = common.MapStr{}
	event.RootFields.Put("service.name", "kibana")

	// Set elasticsearch cluster id
	if clusterID, ok := dataFields["cluster_uuid"]; ok {
		delete(dataFields, "cluster_uuid")
		event.RootFields.Put("elasticsearch.cluster.id", clusterID)
	}

	event.MetricSetFields = dataFields

	r.Event(event)

	return err
}
