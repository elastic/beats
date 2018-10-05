// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package db

import (
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
)

var (
	schema = s.Schema{
		"database": s.Object{
			"id": c.Str("database_id"),
		},
		//Returns space usage information for the transaction log.
		"log_space_usage": s.Object{
			"total_bytes":             c.Int("total_log_size_in_bytes"),
			"used_bytes":              c.Int("used_log_space_in_bytes"),
			"used_percent":            c.Float("used_log_space_in_percent"),
			"since_last_backup_bytes": c.Int("log_space_in_bytes_since_last_backup"),
		},
	}
)
