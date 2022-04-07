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

package info

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
	s "github.com/elastic/beats/v8/libbeat/common/schema"
	c "github.com/elastic/beats/v8/libbeat/common/schema/mapstrstr"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/module/haproxy"

	"reflect"
	"strconv"
	"strings"
)

var (
	schema = s.Schema{
		"threads":            c.Int("Nbthread", s.Optional),
		"processes":          c.Int("Nbproc"),
		"process_num":        c.Int("ProcessNum"),
		"pid":                c.Int("Pid"),
		"ulimit_n":           c.Int("UlimitN"),
		"tasks":              c.Int("Tasks"),
		"run_queue":          c.Int("RunQueue"),
		"stopping":           c.Int("Stopping"),
		"jobs":               c.Int("Jobs"),
		"unstoppable_jobs":   c.Int("UnstoppableJobs", s.Optional),
		"listeners":          c.Int("Listeners", s.Optional),
		"dropped_logs":       c.Int("DroppedLogs", s.Optional),
		"busy_polling":       c.Int("BusyPolling", s.Optional),
		"failed_resolutions": c.Int("FailedResolutions", s.Optional),

		"uptime": s.Object{
			"sec": c.Int("UptimeSec"),
		},

		"memory": s.Object{
			"max": s.Object{
				"bytes": c.Int("MemMax"),
			},
		},

		"peers": s.Object{
			"active":    c.Int("ActivePeers", s.Optional),
			"connected": c.Int("ConnectedPeers", s.Optional),
		},

		"bytes": s.Object{
			"out": s.Object{
				"total": c.Int("TotalBytesOut", s.Optional),
				"rate":  c.Int("BytesOutRate", s.Optional),
			},
		},

		"pool": s.Object{
			"allocated": c.Int("PoolAlloc", s.Optional),
			"used":      c.Int("PoolUsed", s.Optional),
			"failed":    c.Int("PoolFailed", s.Optional),
		},

		"compress": s.Object{
			"bps": s.Object{
				"in":         c.Int("CompressBpsIn"),
				"out":        c.Int("CompressBpsOut"),
				"rate_limit": c.Int("CompressBpsRateLim"),
			},
		},

		"connection": s.Object{
			"rate": s.Object{
				"value": c.Int("ConnRate"),
				"limit": c.Int("ConnRateLimit"),
				"max":   c.Int("MaxConnRate"),
			},
			"ssl": s.Object{
				"current": c.Int("CurrSslConns"),
				"total":   c.Int("CumSslConns"),
				"max":     c.Int("MaxSslConns"),
			},
			"current":  c.Int("CurrConns"),
			"total":    c.Int("CumConns"),
			"hard_max": c.Int("HardMaxconn"),
			"max":      c.Int("Maxconn"),
		},

		"requests": s.Object{
			"total": c.Int("CumReq"),
		},

		"sockets": s.Object{
			"max": c.Int("Maxsock"),
		},

		"pipes": s.Object{
			"used": c.Int("PipesUsed"),
			"free": c.Int("PipesFree"),
			"max":  c.Int("Maxpipes"),
		},

		"session": s.Object{
			"rate": s.Object{
				"value": c.Int("SessRate"),
				"limit": c.Int("SessRateLimit"),
				"max":   c.Int("MaxSessRate"),
			},
		},

		"ssl": s.Object{
			"rate": s.Object{
				"value": c.Int("SslRate"),
				"limit": c.Int("SslRateLimit"),
				"max":   c.Int("MaxSslRate"),
			},
			"frontend": s.Object{
				"key_rate": s.Object{
					"value": c.Int("SslFrontendKeyRate"),
					"max":   c.Int("SslFrontendMaxKeyRate"),
				},
				"session_reuse": s.Object{
					"pct": c.Float("SslFrontendSessionReusePct"),
				},
			},
			"backend": s.Object{
				"key_rate": s.Object{
					"value": c.Int("SslBackendKeyRate"),
					"max":   c.Int("SslBackendMaxKeyRate"),
				},
			},
			"cached_lookups": c.Int("SslCacheLookups"),
			"cache_misses":   c.Int("SslCacheMisses"),
		},

		"zlib_mem_usage": s.Object{
			"value": c.Int("ZlibMemUsage"),
			"max":   c.Int("MaxZlibMemUsage"),
		},

		"idle": s.Object{
			"pct": c.Float("IdlePct"),
		},
	}
)

// Map data to MapStr
func eventMapping(info *haproxy.Info, r mb.ReporterV2) (mb.Event, error) {
	// Full mapping from info

	st := reflect.ValueOf(info).Elem()
	typeOfT := st.Type()
	source := map[string]interface{}{}

	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)

		if typeOfT.Field(i).Name == "IdlePct" {
			// Convert this value to a float between 0.0 and 1.0
			fval, err := strconv.ParseFloat(f.Interface().(string), 64)
			if err != nil {
				return mb.Event{}, errors.Wrap(err, "error getting IdlePct")
			}
			source[typeOfT.Field(i).Name] = strconv.FormatFloat(fval/float64(100), 'f', 2, 64)
		} else if typeOfT.Field(i).Name == "Memmax_MB" {
			// Convert this value to bytes
			val, err := strconv.Atoi(strings.TrimSpace(f.Interface().(string)))
			if err != nil {
				r.Error(err)
				return mb.Event{}, errors.Wrap(err, "error getting Memmax_MB")
			}
			source[typeOfT.Field(i).Name] = strconv.Itoa((val * 1024 * 1024))
		} else {
			if f.Interface().(string) == "" {
				source[typeOfT.Field(i).Name] = "0"
			} else {
				source[typeOfT.Field(i).Name] = f.Interface()
			}
		}

	}

	event := mb.Event{
		RootFields: common.MapStr{},
	}

	fields, err := schema.Apply(source)
	if err != nil {
		return event, errors.Wrap(err, "error applying schema")
	}
	if processID, err := fields.GetValue("pid"); err == nil {
		event.RootFields.Put("process.pid", processID)
		fields.Delete("pid")
	}

	event.MetricSetFields = fields
	return event, nil
}
