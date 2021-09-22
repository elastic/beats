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
		"attributes.kibanaSavedObjectMeta.searchSourceJSON",
		"attributes.layerListJSON",
		"attributes.mapStateJSON",
		"attributes.optionsJSON",
		"attributes.panelsJSON",
		"attributes.uiStateJSON",
		"attributes.visState",
	}
)

// DecodeExported decodes an exported dashboard
func DecodeExported(exported []byte) []byte {
	// remove unsupported chars
	var result bytes.Buffer
	r := bufio.NewReader(bytes.NewReader(exported))
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				_, err = result.Write(decodeLine(line))
				if err != nil {
					return exported
				}
				return result.Bytes()
			}
			return exported
		}
		_, err = result.Write(decodeLine(line))
		if err != nil {
			return exported
		}
		_, err = result.WriteRune('\n')
		if err != nil {
			return exported
		}
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
	o = decodeObject(o)
	o = decodeEmbeddableConfig(o)

	return []byte(o.String())
}

func decodeObject(o common.MapStr) common.MapStr {
	for _, key := range responseToDecode {
		// All fields are optional, so errors are not caught
		err := decodeValue(o, key)
		if err != nil {
			logger := logp.NewLogger("dashboards")
			logger.Debugf("Error while decoding dashboard objects: %+v", err)
			continue
		}
	}

	return o
}

func decodeEmbeddableConfig(o common.MapStr) common.MapStr {
	p, err := o.GetValue("attributes.panelsJSON")
	if err != nil {
		return o
	}

	if panels, ok := p.([]interface{}); ok {
		for i, pan := range panels {
			if panel, ok := pan.(map[string]interface{}); ok {
				panelObj := common.MapStr(panel)
				embedded, err := panelObj.GetValue("embeddableConfig")
				if err != nil {
					continue
				}
				if embeddedConfig, ok := embedded.(map[string]interface{}); ok {
					embeddedConfigObj := common.MapStr(embeddedConfig)
					panelObj.Put("embeddableConfig", decodeObject(embeddedConfigObj))
					panels[i] = panelObj
				}
			}
		}
		o.Put("attributes.panelsJSON", panels)
	}

	return o
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
