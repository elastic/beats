// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stats

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

		"crawler": c.Dict("crawler", s.Schema{
			"global": c.Dict("global", s.Schema{
				"crawl_requests": c.Dict("crawl_requests", s.Schema{
					"pending":    c.Int("pending"),
					"active":     c.Int("active"),
					"successful": c.Int("successful"),
					"failed":     c.Int("failed"),
				}),
			}),

			"node": c.Dict("node", s.Schema{
				"pages_visited": c.Int("pages_visited"),
				"urls_allowed":  c.Int("urls_allowed"),
				"urls_denied": c.Dict("urls_denied", s.Schema{
					"already_seen":             c.Int("already_seen", s.Optional),
					"domain_filter_denied":     c.Int("domain_filter_denied", s.Optional),
					"incorrect_protocol":       c.Int("incorrect_protocol", s.Optional),
					"link_too_deep":            c.Int("link_too_deep", s.Optional),
					"nofollow":                 c.Int("nofollow", s.Optional),
					"unsupported_content_type": c.Int("unsupported_content_type", s.Optional),
				}),

				"status_codes": c.Dict("status_codes", s.Schema{
					"200": c.Int("200", s.Optional),
					"301": c.Int("301", s.Optional),
					"302": c.Int("302", s.Optional),
					"304": c.Int("304", s.Optional),
					"400": c.Int("400", s.Optional),
					"401": c.Int("401", s.Optional),
					"402": c.Int("402", s.Optional),
					"403": c.Int("403", s.Optional),
					"404": c.Int("404", s.Optional),
					"405": c.Int("405", s.Optional),
					"410": c.Int("410", s.Optional),
					"422": c.Int("422", s.Optional),
					"429": c.Int("429", s.Optional),
					"500": c.Int("500", s.Optional),
					"501": c.Int("501", s.Optional),
					"502": c.Int("502", s.Optional),
					"503": c.Int("503", s.Optional),
					"504": c.Int("504", s.Optional),
				}),

				"queue_size": c.Dict("queue_size", s.Schema{
					"primary": c.Int("primary"),
					"purge":   c.Int("purge"),
				}),

				"active_threads": c.Int("active_threads"),
				"workers": c.Dict("workers", s.Schema{
					"pool_size": c.Int("pool_size"),
					"active":    c.Int("active"),
					"available": c.Int("available"),
				}),
			}),
		}),

		"product_usage": c.Dict("product_usage", s.Schema{
			"app_search": c.Dict("app_search", s.Schema{
				"total_engines": c.Int("total_engines"),
			}),
			"workplace_search": c.Dict("workplace_search", s.Schema{
				"total_org_sources":     c.Int("total_org_sources"),
				"total_private_sources": c.Int("total_private_sources"),
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

	// xpack.enabled in config using standalone metricbeat writes to `.monitoring` instead of `metricbeat-*`
	// When using Agent, the index name is overwritten anyways.
	if isXpack {
		index := elastic.MakeXPackMonitoringIndexName(elastic.EnterpriseSearch)
		event.Index = index
	}

	event.MetricSetFields, err = schema.Apply(data)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "failure to apply stats schema"))
	} else {
		report.Event(event)
	}

	return errs.Err()
}
