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

package asset

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"io/ioutil"
	"sort"
)

// FieldsRegistry contains a list of fields.yml files
// As each entry is an array of bytes multiple fields.yml can be added under one path.
// This can become useful as we don't have to generate anymore the fields.yml but can
// package each local fields.yml from things like processors.
var FieldsRegistry = map[string]map[int]map[string]string{}

// SetFields sets the fields for a given beat and asset name
func SetFields(beat, name string, p Priority, asset func() string) error {
	data := asset()

	priority := int(p)

	if _, ok := FieldsRegistry[beat]; !ok {
		FieldsRegistry[beat] = map[int]map[string]string{}
	}

	if _, ok := FieldsRegistry[beat][priority]; !ok {
		FieldsRegistry[beat][priority] = map[string]string{}
	}

	FieldsRegistry[beat][priority][name] = data

	return nil
}

// GetFields returns a byte array contains all fields for the given beat
func GetFields(beat string) ([]byte, error) {
	var fields []byte

	// Get all priorities and sort them
	beatRegistry := FieldsRegistry[beat]
	priorities := make([]int, 0, len(beatRegistry))
	for p := range beatRegistry {
		priorities = append(priorities, p)
	}
	sort.Ints(priorities)

	for _, priority := range priorities {

		priorityRegistry := beatRegistry[priority]

		// Sort all entries with same priority alphabetically
		entries := make([]string, 0, len(priorityRegistry))
		for e := range priorityRegistry {
			entries = append(entries, e)
		}
		sort.Strings(entries)

		for _, entry := range entries {
			data := priorityRegistry[entry]
			output, err := DecodeData(data)
			if err != nil {
				return nil, err
			}

			fields = append(fields, output...)
		}
	}
	return fields, nil
}

// EncodeData compresses the data with zlib and base64 encodes it
func EncodeData(data string) (string, error) {
	var zlibBuf bytes.Buffer
	writer := zlib.NewWriter(&zlibBuf)
	_, err := writer.Write([]byte(data))
	if err != nil {
		return "", err
	}
	err = writer.Close()
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(zlibBuf.Bytes()), nil
}

// DecodeData base64 decodes the data and uncompresses it
func DecodeData(data string) ([]byte, error) {

	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	b := bytes.NewReader(decoded)
	r, err := zlib.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return ioutil.ReadAll(r)
}
