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

package store

import (
	"encoding/json"

	"github.com/menderesk/beats/v7/libbeat/common"

	s "github.com/menderesk/beats/v7/libbeat/common/schema"
	c "github.com/menderesk/beats/v7/libbeat/common/schema/mapstriface"
)

var (
	schema = s.Schema{
		"gets": s.Object{
			"success": c.Int("getsSuccess"),
			"fail":    c.Int("getsFail"),
		},
		"sets": s.Object{
			"success": c.Int("setsSuccess"),
			"fail":    c.Int("setsFail"),
		},
		"delete": s.Object{
			"success": c.Int("deleteSuccess"),
			"fail":    c.Int("deleteFail"),
		},
		"update": s.Object{
			"success": c.Int("updateSuccess"),
			"fail":    c.Int("updateFail"),
		},
		"create": s.Object{
			"success": c.Int("createSuccess"),
			"fail":    c.Int("createFail"),
		},
		"compareandswap": s.Object{
			"success": c.Int("compareAndSwapSuccess"),
			"fail":    c.Int("compareAndSwapFail"),
		},
		"compareanddelete": s.Object{
			"success": c.Int("compareAndDeleteSuccess"),
			"fail":    c.Int("compareAndDeleteFail"),
		},
		"expire": s.Object{
			"count": c.Int("expireCount"),
		},
		"watchers": c.Int("watchers"),
	}
)

func eventMapping(content []byte) common.MapStr {
	var data map[string]interface{}
	json.Unmarshal(content, &data)
	event, _ := schema.Apply(data)
	return event
}
