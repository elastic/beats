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
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/helper/elastic"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	schemaXPackMonitoringStats = s.Schema{
		"concurrent_connections": c.Int("concurrent_connections"),
		"os": c.Dict("os", s.Schema{
			"load": c.Dict("load", s.Schema{
				"1m":  c.Float("1m"),
				"5m":  c.Float("5m"),
				"15m": c.Float("15m"),
			}),
			"memory": c.Dict("memory", s.Schema{
				"total_in_bytes": c.Int("total_bytes"),
				"free_in_bytes":  c.Int("free_bytes"),
				"used_in_bytes":  c.Int("used_bytes"),
			}),
			"uptime_in_millis": c.Int("uptime_ms"),
			"distro":           c.Str("distro", s.Optional),
			"distroRelease":    c.Str("distro_release", s.Optional),
			"platform":         c.Str("platform", s.Optional),
			"platformRelease":  c.Str("platform_release", s.Optional),
		}),
		"process": c.Dict("process", s.Schema{
			"event_loop_delay": c.Float("event_loop_delay"),
			"memory": c.Dict("memory", s.Schema{
				"heap": c.Dict("heap", s.Schema{
					"total_in_bytes": c.Int("total_bytes"),
					"used_in_bytes":  c.Int("used_bytes"),
					"size_limit":     c.Int("size_limit"),
				}),
			}),
			"uptime_in_millis": c.Int("uptime_ms"),
		}),
		"requests": RequestsDict,
		"response_times": c.Dict("response_times", s.Schema{
			"average": c.Int("avg_ms", s.Optional),
			"max":     c.Int("max_ms", s.Optional),
		}, c.DictOptional),
		"kibana": c.Dict("kibana", s.Schema{
			"uuid":              c.Str("uuid"),
			"name":              c.Str("name"),
			"index":             c.Str("index"),
			"host":              c.Str("host"),
			"transport_address": c.Str("transport_address"),
			"version":           c.Str("version"),
			"snapshot":          c.Bool("snapshot"),
			"status":            c.Str("status"),
		}),
	}

	reportingCsvDict = c.Dict("csv", s.Schema{
		"available": c.Bool("available"),
		"total":     c.Int("total"),
	}, c.DictOptional)

	reportingPrintablePdfDict = c.Dict("printable_pdf", s.Schema{
		"available": c.Bool("available"),
		"total":     c.Int("total"),
		"app": c.Dict("app", s.Schema{
			"visualization": c.Int("visualization"),
			"dashboard":     c.Int("dashboard"),
		}, c.DictOptional),
		"layout": c.Dict("layout", s.Schema{
			"print":           c.Int("print"),
			"preserve_layout": c.Int("preserve_layout"),
		}, c.DictOptional),
	}, c.DictOptional)

	reportingStatusDict = c.Dict("status", s.Schema{
		"completed":  c.Int("completed", s.Optional),
		"failed":     c.Int("failed", s.Optional),
		"processing": c.Int("processing", s.Optional),
		"pending":    c.Int("pending", s.Optional),
	}, c.DictOptional)

	reportingPeriodSchema = s.Schema{
		"_all":          c.Int("all"),
		"csv":           reportingCsvDict,
		"printable_pdf": reportingPrintablePdfDict,
		"status":        reportingStatusDict,
	}
)

type dataParser func(mb.ReporterV2, common.MapStr, time.Time) (string, string, common.MapStr, error)

func statsDataParser(r mb.ReporterV2, data common.MapStr, now time.Time) (string, string, common.MapStr, error) {
	clusterUUID, ok := data["clusterUuid"].(string)
	if !ok {
		return "", "", nil, elastic.MakeErrorForMissingField("clusterUuid", elastic.Kibana)
	}

	kibanaStatsFields, err := schemaXPackMonitoringStats.Apply(data)
	if err != nil {
		return "", "", nil, err
	}

	process, ok := data["process"].(map[string]interface{})
	if !ok {
		return "", "", nil, elastic.MakeErrorForMissingField("process", elastic.Kibana)
	}
	memory, ok := process["memory"].(map[string]interface{})
	if !ok {
		return "", "", nil, elastic.MakeErrorForMissingField("process.memory", elastic.Kibana)
	}
	rss, ok := memory["resident_set_size_bytes"].(float64)
	if !ok {
		return "", "", nil, elastic.MakeErrorForMissingField("process.memory.resident_set_size_bytes", elastic.Kibana)
	}
	kibanaStatsFields.Put("process.memory.resident_set_size_in_bytes", int64(rss))

	kibanaStatsFields.Put("timestamp", now)

	// Make usage field passthrough as-is
	usage, ok := data["usage"].(map[string]interface{})
	if !ok {
		return "", "", nil, elastic.MakeErrorForMissingField("usage", elastic.Kibana)
	}
	kibanaStatsFields.Put("usage", usage)

	return "kibana_stats", clusterUUID, kibanaStatsFields, nil
}

func settingsDataParser(r mb.ReporterV2, data common.MapStr, now time.Time) (string, string, common.MapStr, error) {
	clusterUUID, ok := data["cluster_uuid"].(string)
	if !ok {
		return "", "", nil, elastic.MakeErrorForMissingField("cluster_uuid", elastic.Kibana)
	}

	kibanaSettingsFields, ok := data["settings"]
	if !ok {
		return "", "", nil, elastic.MakeErrorForMissingField("settings", elastic.Kibana)
	}

	return "kibana_settings", clusterUUID, kibanaSettingsFields.(map[string]interface{}), nil
}

func eventMappingXPack(r mb.ReporterV2, intervalMs int64, now time.Time, content []byte, dataParserFunc dataParser) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Kibana API response")
	}

	t, clusterUUID, fields, err := dataParserFunc(r, data, now)
	if err != nil {
		return errors.Wrap(err, "failure to parse data")
	}

	var event mb.Event
	event.RootFields = common.MapStr{
		"cluster_uuid": clusterUUID,
		"timestamp":    now,
		"interval_ms":  intervalMs,
		"type":         t,
		t:              fields,
	}

	event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Kibana)

	r.Event(event)
	return nil
}

func eventMappingStatsXPack(r mb.ReporterV2, intervalMs int64, now time.Time, content []byte) error {
	return eventMappingXPack(r, intervalMs, now, content, statsDataParser)
}

func eventMappingSettingsXPack(r mb.ReporterV2, intervalMs int64, now time.Time, content []byte) error {
	return eventMappingXPack(r, intervalMs, now, content, settingsDataParser)
}
