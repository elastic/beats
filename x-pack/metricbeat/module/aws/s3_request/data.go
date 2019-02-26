// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3_request

import (
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
)

var (
	schemaMetricSetFields = s.Schema{
		"all_requests":    c.Int("AllRequests"),
		"get_requests":    c.Int("GetRequests"),
		"put_requests":    c.Int("PutRequests"),
		"delete_requests": c.Int("DeleteRequests"),
		"head_requests":   c.Int("HeadRequests"),
		"post_requests":   c.Int("PostRequests"),
		"select_requests": c.Int("SelectRequests"),
		"select_scanned": s.Object{
			"bytes": c.Float("SelectScannedBytes"),
		},
		"select_returned": s.Object{
			"bytes": c.Float("SelectReturnedBytes"),
		},
		"list_requests":         c.Int("ListRequests"),
		"bytes_downloaded":      c.Float("BytesDownloaded"),
		"bytes_uploaded":        c.Float("BytesUploaded"),
		"4xx_errors":            c.Int("4xxErrors"),
		"5xx_errors":            c.Int("5xxErrors"),
		"first_byte_latency":    c.Float("FirstByteLatency"),
		"total_request_latency": c.Float("TotalRequestLatency"),
	}
)
