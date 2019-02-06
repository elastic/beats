// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3

import (
	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
)

var (
	schemaMetricSetFields = s.Schema{
		"bucket": s.Object{
			"name": c.Str("bucket.name", s.Optional),
			"storage": s.Object{
				"type": c.Str("bucket.storage.type", s.Optional),
			},
			"size": s.Object{
				"bytes": c.Float("BucketSizeBytes", s.Optional),
			},
			"head_requests":         c.Int("HeadRequests", s.Optional),
			"all_requests":          c.Int("AllRequests", s.Optional),
			"4xx_errors":            c.Int("4xxErrors", s.Optional),
			"5xx_errors":            c.Int("5xxErrors", s.Optional),
			"first_byte_latency":    c.Float("FirstByteLatency", s.Optional),
			"list_requests":         c.Int("ListRequests", s.Optional),
			"bytes_downloaded":      c.Float("BytesDownloaded", s.Optional),
			"total_request_latency": c.Float("TotalRequestLatency", s.Optional),
		},
		"object": s.Object{
			"count": c.Int("NumberOfObjects", s.Optional),
		},
	}
)

var (
	schemaRootFields = s.Schema{
		"service": s.Object{
			"name": c.Str("service.name", s.Optional),
		},
		"cloud": s.Object{
			"provider": c.Str("cloud.provider", s.Optional),
			"region":   c.Str("cloud.region", s.Optional),
		},
	}
)

func eventMapping(input map[string]interface{}, schema s.Schema) (common.MapStr, error) {
	return schema.Apply(input)
}
