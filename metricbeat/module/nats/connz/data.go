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

package connz

import (
	"encoding/json"
	"github.com/elastic/beats/libbeat/common"
	"time"
)

type Connz struct {
	ServerID       string        `json:"server_id"`
	Now            time.Time     `json:"now"`
	NumConnections int           `json:"num_connections"`
	Total          int           `json:"total"`
	Offset         int           `json:"offset"`
	Limit          int           `json:"limit"`
	Connections    []interface{} `json:"connections"`
}

func eventMapping(content []byte) common.MapStr {
	var data Connz
	json.Unmarshal(content, &data)
	// TODO: add error handling
	event := common.MapStr{
		"metrics": data,
	}
	return event
}
