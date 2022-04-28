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

package kibana

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// RemoveIndexPattern removes the index pattern entry from a given dashboard export
func RemoveIndexPattern(data []byte) ([]byte, error) {
	var result []byte
	r := bufio.NewReader(bytes.NewReader(data))
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				res, removeErr := removeLineIfIndexPattern(line)
				if removeErr != nil {
					return data, removeErr
				}
				return append(result, res...), nil
			}
			return data, err
		}

		res, err := removeLineIfIndexPattern(line)
		if err != nil {
			return data, err
		}
		result = append(result, res...)
	}
}

func removeLineIfIndexPattern(line []byte) ([]byte, error) {
	if len(bytes.TrimSpace(line)) == 0 {
		return line, nil
	}

	var r mapstr.M
	// Full struct need to not loose any data
	err := json.Unmarshal(line, &r)
	if err != nil {
		return nil, err
	}
	v, err := r.GetValue("type")
	if err != nil {
		return nil, fmt.Errorf("type key not found or not string")
	}
	if v != "index-pattern" {
		return line, nil
	}

	return nil, nil
}
