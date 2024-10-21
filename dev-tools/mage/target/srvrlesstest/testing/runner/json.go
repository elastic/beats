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

package runner

import (
	"bufio"
	"bytes"
	"encoding/json"
)

type JSONTestEntry struct {
	Time    string `json:"Time"`
	Action  string `json:"Action"`
	Package string `json:"Package"`
	Test    string `json:"Test"`
	Output  string `json:"Output"`
}

func suffixJSONResults(content []byte, suffix string) ([]byte, error) {
	var result bytes.Buffer
	sc := bufio.NewScanner(bytes.NewReader(content))
	for sc.Scan() {
		var entry JSONTestEntry
		err := json.Unmarshal([]byte(sc.Text()), &entry)
		if err != nil {
			return nil, err
		}
		if entry.Package != "" {
			entry.Package += suffix
		}
		raw, err := json.Marshal(&entry)
		if err != nil {
			return nil, err
		}
		_, err = result.Write(raw)
		if err != nil {
			return nil, err
		}
		_, err = result.Write([]byte("\n"))
		if err != nil {
			return nil, err
		}
	}
	return result.Bytes(), nil
}
