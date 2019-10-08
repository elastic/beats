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

package pipeline

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// UnmarshalPipeline unmarshals a JSON or YAML formatted ingest pipeline.
func UnmarshalPipeline(filePath string, fileContents []byte) (map[string]interface{}, error) {
	var content map[string]interface{}
	switch extension := strings.ToLower(filepath.Ext(filePath)); extension {
	case ".json":
		if err := json.Unmarshal(fileContents, &content); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal the JSON pipeline file '%s'", filePath)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(fileContents, &content); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal the YAML pipeline file '%s'", filePath)
		}
		newContent, err := fixYAMLMaps(content)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to sanitize the YAML pipeline file '%s'", filePath)
		}
		content = newContent.(map[string]interface{})
	default:
		return nil, fmt.Errorf("Unsupported extension '%s' for pipeline file '%s'", extension, filePath)
	}

	return content, nil
}

// This function recursively converts maps with interface{} keys, as returned by
// yaml.Unmarshal, to maps of string keys, as expected by the json encoder
// that will be used when delivering the pipeline to Elasticsearch.
// Will return an error when something other than a string is used as a key.
func fixYAMLMaps(elem interface{}) (_ interface{}, err error) {
	switch v := elem.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, value := range v {
			keyS, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("key '%v' is not string but %T", key, key)
			}
			if result[keyS], err = fixYAMLMaps(value); err != nil {
				return nil, err
			}
		}
		return result, nil
	case map[string]interface{}:
		for key, value := range v {
			if v[key], err = fixYAMLMaps(value); err != nil {
				return nil, err
			}
		}
	case []interface{}:
		for idx, value := range v {
			if v[idx], err = fixYAMLMaps(value); err != nil {
				return nil, err
			}
		}
	}
	return elem, nil
}
