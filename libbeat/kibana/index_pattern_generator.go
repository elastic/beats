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

	"github.com/elastic/beats/libbeat/common"
)

type IndexPatternGenerator struct {
	indexName      string
	beatVersion    string
	fieldsYaml     string
	version        common.Version
	targetDir      string
	targetFilename string
}

// Create an instance of the Kibana Index Pattern Generator
func NewGenerator(indexName, beatName, fieldsYAMLFile, outputDir, beatVersion string, version common.Version) (*IndexPatternGenerator, error) {
	beatName = clean(beatName)

	if _, err := os.Stat(fieldsYAMLFile); err != nil {
		return nil, err
	}

	return &IndexPatternGenerator{
		indexName:      indexName,
		fieldsYaml:     fieldsYAMLFile,
		beatVersion:    beatVersion,
		version:        version,
		targetDir:      createTargetDir(outputDir, version),
		targetFilename: beatName + ".json",
	}, nil
}

// Create the Index-Pattern for Kibana for 5.x and default.
func (i *IndexPatternGenerator) Generate() (string, error) {
	idxPattern, err := i.generate()
	if err != nil {
		return "", err
	}

	if i.version.Major >= 6 {
		idxPattern = i.generateMinVersion6(idxPattern)
	}

	file := filepath.Join(i.targetDir, i.targetFilename)
	err = dumpToFile(file, idxPattern)

	return file, err
}

func (i *IndexPatternGenerator) generate() (common.MapStr, error) {
	indexPattern := common.MapStr{
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

func (i *IndexPatternGenerator) generateMinVersion6(attrs common.MapStr) common.MapStr {
	out := common.MapStr{
		"version": i.beatVersion,
		"objects": []common.MapStr{
			common.MapStr{
				"type":       "index-pattern",
				"id":         i.indexName,
				"version":    "1",
				"attributes": attrs,
			},
		},
	}

	return out
}

func (i *IndexPatternGenerator) addGeneral(indexPattern *common.MapStr) error {
	kibanaEntries, err := loadKibanaEntriesFromYaml(i.fieldsYaml)
	if err != nil {
		return err
	}
	transformed := newTransformer(kibanaEntries).transform()
	if srcFilters, ok := transformed["sourceFilters"].([]common.MapStr); ok {
		sourceFiltersBytes, err := json.Marshal(srcFilters)
		if err != nil {
			return err
		}
		(*indexPattern)["sourceFilters"] = string(sourceFiltersBytes)
	}
	return nil
}

func (i *IndexPatternGenerator) addFieldsSpecific(indexPattern *common.MapStr) error {
	fields, err := common.LoadFieldsYaml(i.fieldsYaml)
	if err != nil {
		return err
	}
	transformer, err := newFieldsTransformer(&i.version, fields)
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

func dumpToFile(f string, pattern common.MapStr) error {
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

func createTargetDir(baseDir string, version common.Version) string {
	targetDir := filepath.Join(baseDir, getVersionPath(version), "index-pattern")
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		os.MkdirAll(targetDir, 0755)
	}
	return targetDir
}

func getVersionPath(version common.Version) string {
	versionPath := "6"
	if version.Major == 5 {
		versionPath = "5"
	}
	return versionPath
}
