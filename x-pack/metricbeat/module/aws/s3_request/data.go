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
		"requests": s.Object{
			"total":                 c.Int("AllRequests"),
			"get":                   c.Int("GetRequests"),
			"put":                   c.Int("PutRequests"),
			"delete":                c.Int("DeleteRequests"),
			"head":                  c.Int("HeadRequests"),
			"post":                  c.Int("PostRequests"),
			"select":                c.Int("SelectRequests"),
			"select_scanned.bytes":  c.Float("SelectScannedBytes"),
			"select_returned.bytes": c.Float("SelectReturnedBytes"),
			"list":                  c.Int("ListRequests"),
		},
		"downloaded": s.Object{
			"bytes": c.Float("BytesDownloaded"),
		},
		"uploaded": s.Object{
			"bytes": c.Float("BytesUploaded"),
		},
		"errors": s.Object{
			"4xx": c.Int("4xxErrors"),
			"5xx": c.Int("5xxErrors"),
		},
		"latency": s.Object{
			"first_byte.ms":    c.Float("FirstByteLatency"),
			"total_request.ms": c.Float("TotalRequestLatency"),
		},
	}
)
