// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3_daily_storage

import (
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstrstr"
)

var (
	schemaMetricSetFields = s.Schema{
		"bucket": s.Object{
			"size": s.Object{
				"bytes": c.Float("BucketSizeBytes"),
			},
		},
		"number_of_objects": c.Float("NumberOfObjects"),
	}
)
