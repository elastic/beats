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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
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
func DecodeExported(exported []byte) []byte {
	// remove unsupported chars
	var result []byte
	r := bufio.NewReader(bytes.NewReader(exported))
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return append(result, decodeLine(line)...)
			}
			return exported
		}
		result = append(result, decodeLine(line)...)
	}
}

func decodeLine(line []byte) []byte {
	if len(bytes.TrimSpace(line)) == 0 {
		return line
	}

	o := common.MapStr{}
	err := json.Unmarshal(line, &o)
	if err != nil {
		return line
	}
	var result []byte
	for _, key := range responseToDecode {
		// All fields are optional, so errors are not caught
		err := decodeValue(o, key)
		if err != nil {
			logger := logp.NewLogger("dashboards")
			logger.Debugf("Error while decoding dashboard objects: %+v", err)
		}
		result = append(result, []byte(o.String())...)
	}
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
