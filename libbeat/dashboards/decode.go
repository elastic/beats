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

package dashboards

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

var (
	responseToDecode = []string{
		"attributes.uiStateJSON",
		"attributes.visState",
		"attributes.optionsJSON",
		"attributes.panelsJSON",
		"attributes.kibanaSavedObjectMeta.searchSourceJSON",
	}
)

// DecodeExported decodes an exported dashboard
func DecodeExported(result common.MapStr) common.MapStr {
	// remove unsupported chars
	objects := result["objects"].([]interface{})
	for _, obj := range objects {
		o := obj.(common.MapStr)
		for _, key := range responseToDecode {
			// All fields are optional, so errors are not caught
			err := decodeValue(o, key)
			if err != nil {
				logp.Debug("dashboards", "Error while decoding dashboard objects: %+v", err)
			}
		}
	}
	result["objects"] = objects
	return result
}

func decodeValue(data common.MapStr, key string) error {
	v, err := data.GetValue(key)
	if err != nil {
		return err
	}
	s := v.(string)
	var d interface{}
	err = json.Unmarshal([]byte(s), &d)
	if err != nil {
		return fmt.Errorf("error decoding %s: %v", key, err)
	}

	data.Put(key, d)
	return nil
}
