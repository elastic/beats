// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package ilm

import "github.com/elastic/beats/libbeat/common"

var policies = common.MapStr{
	"default":           beatDefaultPolicy,
	"deleteAfter10Days": deleteAfterTenDays,
	"deleteAfter1Year":  deleteAfterOneYear,
}

var beatDefaultPolicy = common.MapStr{
	"policy": common.MapStr{
		"phases": common.MapStr{
			"hot": common.MapStr{
				"actions": common.MapStr{
					"rollover": common.MapStr{
						"max_size": "50gb",
						"max_age":  "30d",
					},
				},
			},
		},
	},
}

var deleteAfterTenDays = common.MapStr{
	"policy": common.MapStr{
		"phases": common.MapStr{
			"hot": common.MapStr{
				"actions": common.MapStr{
					"rollover": common.MapStr{
						"max_size": "50gb",
						"max_age":  "1d",
					},
				},
			},
			"delete": common.MapStr{
				"min_age": "10d",
				"actions": common.MapStr{
					"delete": common.MapStr{},
				},
			},
		},
	},
}

var deleteAfterOneYear = common.MapStr{
	"policy": common.MapStr{
		"phases": common.MapStr{
			"hot": common.MapStr{
				"actions": common.MapStr{
					"rollover": common.MapStr{
						"max_size": "50gb",
						"max_age":  "1w",
					},
				},
			},
			"delete": common.MapStr{
				"min_age": "1y",
				"actions": common.MapStr{
					"delete": common.MapStr{},
				},
			},
		},
	},
}
