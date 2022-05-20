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
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/elastic/beats/v7/libbeat/mapping"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/version"
)

type IndexPatternGenerator struct {
	indexName   string
	beatVersion string
	fields      []byte
	version     version.V
	migration   bool
}

// Create an instance of the Kibana Index Pattern Generator
func NewGenerator(indexName, beatName string, fields []byte, beatVersion string, v version.V, migration bool) (*IndexPatternGenerator, error) {
	beatName = clean(beatName)

	return &IndexPatternGenerator{
		indexName:   indexName + "-*",
		fields:      fields,
		beatVersion: beatVersion,
		version:     v,
		migration:   migration,
	}, nil
}

// Generate creates the Index-Pattern for Kibana.
func (i *IndexPatternGenerator) Generate() (mapstr.M, error) {
	idxPattern, err := i.generate()
	if err != nil {
		return nil, err
	}

	return i.generatePattern(idxPattern), nil
}

func (i *IndexPatternGenerator) generate() (mapstr.M, error) {
	indexPattern := mapstr.M{
		"timeFieldName": "@timestamp",
		"title":         i.indexName,
	}

	err := i.addGeneral(&indexPattern)
	if err != nil {
		return nil, err
	}

	err = i.addFieldsSpecific(&indexPattern)
	if err != nil {
		return nil, err
	}

	return indexPattern, nil
}

func (i *IndexPatternGenerator) generatePattern(attrs mapstr.M) mapstr.M {
	out := mapstr.M{
		"type":       "index-pattern",
		"id":         i.indexName,
		"version":    i.beatVersion,
		"attributes": attrs,
	}

	return out
}

func (i *IndexPatternGenerator) addGeneral(indexPattern *mapstr.M) error {
	kibanaEntries, err := loadKibanaEntriesFromYaml(i.fields)
	if err != nil {
		return err
	}
	transformed := newTransformer(kibanaEntries).transform()
	if srcFilters, ok := transformed["sourceFilters"].([]mapstr.M); ok {
		sourceFiltersBytes, err := json.Marshal(srcFilters)
		if err != nil {
			return err
		}
		(*indexPattern)["sourceFilters"] = string(sourceFiltersBytes)
	}
	return nil
}

func (i *IndexPatternGenerator) addFieldsSpecific(indexPattern *mapstr.M) error {
	fields, err := mapping.LoadFields(i.fields)
	if err != nil {
		return err
	}
	transformer, err := newFieldsTransformer(&i.version, fields, i.migration)
	if err != nil {
		return err
	}
	transformed, err := transformer.transform()
	if err != nil {
		return err
	}

	fieldsBytes, err := json.Marshal(transformed["fields"])
	if err != nil {
		return err
	}
	(*indexPattern)["fields"] = string(fieldsBytes)

	fieldFormatBytes, err := json.Marshal(transformed["fieldFormatMap"])
	if err != nil {
		return err
	}
	(*indexPattern)["fieldFormatMap"] = string(fieldFormatBytes)
	return nil
}

func clean(name string) string {
	reg := regexp.MustCompile("[^a-zA-Z0-9_]+")
	return reg.ReplaceAllString(name, "")
}

func dumpToFile(f string, pattern mapstr.M) error {
	patternIndent, err := json.MarshalIndent(pattern, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(f, patternIndent, 0644)
	if err != nil {
		return err
	}
	return nil
}

func createTargetDir(baseDir string, version version.V) string {
	targetDir := filepath.Join(baseDir, getVersionPath(version), "index-pattern")
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		os.MkdirAll(targetDir, 0755)
	}
	return targetDir
}

func getVersionPath(version version.V) string {
	return "7"
}
