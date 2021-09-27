// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stats

import (
	"encoding/json"
	"errors"

	"github.com/elastic/beats/v7/libbeat/common"
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
)

var (
	connectorsPoolSchema = s.Schema{
		"queue_depth":     c.Int("queue_depth"),
		"size":            c.Int("size"),
		"busy":            c.Int("busy"),
		"idle":            c.Int("idle"),
		"total_scheduled": c.Int("total_scheduled"),
		"total_completed": c.Int("total_scheduled"),
	}

	schema = s.Schema{
		"http": c.Dict("http", s.Schema{
			"connections": c.Dict("connections", s.Schema{
				"current": c.Int("current"),
				"max":     c.Int("max"),
				"total":   c.Int("total"),
			}),

			"request_duration": c.Dict("request_duration_ms", s.Schema{
				"max.msec":     c.Int("max"),
				"mean.msec":    c.Int("mean"),
				"std_dev.msec": c.Int("std_dev"),
			}),

			"network": c.Dict("network_bytes", s.Schema{
				"received.bytes":         c.Int("received_total"),
				"received.bytes_per_sec": c.Int("received_rate"),
				"sent.bytes":             c.Int("sent_total"),
				"sent.bytes_per_sec":     c.Int("sent_rate"),
			}),

			"responses": c.Dict("responses", s.Schema{
				"1xx": c.Int("1xx"),
				"2xx": c.Int("2xx"),
				"3xx": c.Int("3xx"),
				"4xx": c.Int("4xx"),
				"5xx": c.Int("5xx"),
			}),
		}),

		"queues": c.Dict("queues", s.Schema{
			"engine_destroyer.count": c.Int("engine_destroyer.pending"),
			"process_crawl.count":    c.Int("process_crawl.pending"),
			"mailer.count":           c.Int("mailer.pending"),
			"failed.count":           c.Int("failed.count"),
		}),

		"connectors": c.Dict("connectors", s.Schema{
			"pool": c.Dict("pool", s.Schema{
				"extract_worker_pool":    c.Dict("extract_worker_pool", connectorsPoolSchema),
				"subextract_worker_pool": c.Dict("subextract_worker_pool", connectorsPoolSchema),
				"publish_worker_pool":    c.Dict("publish_worker_pool", connectorsPoolSchema),
			}),

			"job_store": c.Dict("job_store", s.Schema{
				"waiting": c.Int("waiting"),
				"working": c.Int("working"),
				"job_types": c.Dict("job_types", s.Schema{
					"full":        c.Int("full"),
					"incremental": c.Int("incremental"),
					"delete":      c.Int("delete"),
					"permissions": c.Int("permissions"),
				}),
			}),
		}),
	}
)

func eventMapping(input []byte) (common.MapStr, error) {
	var data map[string]interface{}
	err := json.Unmarshal(input, &data)
	if err != nil {
		return nil, err
	}

	// Get queues information
	queues, ok := data["queues"].(map[string]interface{})
	if !ok {
		return nil, errors.New("queues is not a map")
	}

	// Get the list of failed items
	failed, ok := queues["failed"].([]interface{})
	if !ok {
		return nil, errors.New("queues.failed is not an array of maps")
	}

	// Generate a failes items count to be used as a metric
	queues["failed.count"] = len(failed)

	dataFields, err := schema.Apply(data)
	return dataFields, err
}
