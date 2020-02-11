// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package database_account

import (
	"github.com/elastic/beats/libbeat/common"
)

var (
	missingResourcesConfig = common.MapStr{
		"module":          "azure",
		"period":          "60s",
		"metricsets":      []string{"database_account"},
		"client_secret":   "unique identifier",
		"client_id":       "unique identifier",
		"subscription_id": "unique identifier",
		"tenant_id":       "unique identifier",
	}

	resourceConfig = common.MapStr{
		"module":          "azure",
		"period":          "60s",
		"metricsets":      []string{"database_account"},
		"client_secret":   "unique identifier",
		"client_id":       "unique identifier",
		"subscription_id": "unique identifier",
		"tenant_id":       "unique identifier",
		"resources": []common.MapStr{
			{
				"resource_id": "test",
				"metrics": []map[string]interface{}{
					{
						"name": []string{"*"},
					}},
			}},
	}
)
