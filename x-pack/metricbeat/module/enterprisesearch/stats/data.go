// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stats

import (
	"encoding/json"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

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
		"cluster_uuid": c.Str("cluster_uuid"), // This is going to be included in 7.16+
		"http": c.Dict("http", s.Schema{
			"connections": c.Dict("connections", s.Schema{
				"current": c.Int("current"),
				"max":     c.Int("max"),
				"total":   c.Int("total"),
			}),

			"request_duration": c.Dict("request_duration_ms", s.Schema{
				"max":     s.Object{"ms": c.Int("max")},
				"mean":    s.Object{"ms": c.Int("mean")},
				"std_dev": s.Object{"ms": c.Int("std_dev")},
			}),

			"network": c.Dict("network_bytes", s.Schema{
				"received": s.Object{
					"bytes":         c.Int("received_total"),
					"bytes_per_sec": c.Int("received_rate"),
				},
				"sent": s.Object{
					"bytes":         c.Int("sent_total"),
					"bytes_per_sec": c.Int("sent_rate"),
				},
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
			"engine_destroyer": s.Object{"count": c.Int("engine_destroyer.pending")},
			"process_crawl":    s.Object{"count": c.Int("process_crawl.pending")},
			"mailer":           s.Object{"count": c.Int("mailer.pending")},
			"failed":           s.Object{"count": c.Int("failed.count")},
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
	var errs multierror.Errors

	// Get queues information
	queues, ok := data["queues"].(map[string]interface{})
	if ok {
		// Get the list of failed items
		failed, ok := queues["failed"].([]interface{})
		if ok {
			// Use the failed items count as a metric
			queues["failed.count"] = len(failed)
		} else {
			errs = append(errs, errors.New("queues.failed is not an array of maps"))
		}
	} else {
		errs = append(errs, errors.New("queues is not a map"))
	}

	dataFields, err := schema.Apply(data)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "failure to apply stats schema"))
	}
	return dataFields, errs.Err()
}
