// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3_request

import (
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
)

var (
	schemaRequestFields = s.Schema{
		"bucket": s.Object{
			"name":                  c.Str("bucket.name", s.Optional),
			"filter_id":             c.Str("bucket.filter_id", s.Optional),
			"all_requests":          c.Int("AllRequests", s.Optional),
			"get_requests":          c.Int("GetRequests", s.Optional),
			"put_requests":          c.Int("PutRequests", s.Optional),
			"delete_requests":       c.Int("DeleteRequests", s.Optional),
			"head_requests":         c.Int("HeadRequests", s.Optional),
			"post_requests":         c.Int("PostRequests", s.Optional),
			"select_requests":       c.Int("SelectRequests", s.Optional),
			"select_scanned_bytes":  c.Int("SelectScannedBytes", s.Optional),
			"select_returned_bytes": c.Int("SelectReturnedBytes", s.Optional),
			"list_requests":         c.Int("ListRequests", s.Optional),
			"bytes_downloaded":      c.Float("BytesDownloaded", s.Optional),
			"bytes_uploaded":        c.Int("BytesUploaded", s.Optional),
			"4xx_errors":            c.Int("4xxErrors", s.Optional),
			"5xx_errors":            c.Int("5xxErrors", s.Optional),
			"first_byte_latency":    c.Float("FirstByteLatency", s.Optional),
			"total_request_latency": c.Float("TotalRequestLatency", s.Optional),
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
