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

package fields

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

const (
	pipelinePath  = "%s/module/%s/%s/ingest/pipeline.json"
	fieldsYmlPath = "%s/module/%s/%s/_meta/fields.yml"

	typeIdx     = 0
	elementsIdx = 1
	hintIdx     = 2
)

var (
	types = map[string]string{
		"group":           "group",
		"DATA":            "text",
		"GREEDYDATA":      "text",
		"GREEDYMULTILINE": "text",
		"HOSTNAME":        "keyword",
		"IP":              "ip",
		"IPV4":            "ip",
		"IPV6":            "ip",
		"IPHOST":          "keyword",
		"IPORHOST":        "keyword",
		"LOGLEVEL":        "keyword",
		"MULTILINEQUERY":  "text",
		"NUMBER":          "long",
		"POSINT":          "long",
		"SYSLOGHOST":      "keyword",
		"SYSLOGTIMESTAMP": "text",
		"LOCALDATETIME":   "text",
		"TIMESTAMP":       "text",
		"USERNAME":        "keyword",
		"WORD":            "keyword",
	}
)

type pipeline struct {
	Description string                   `json:"description"`
	Processors  []map[string]interface{} `json:"processors"`
	OnFailure   interface{}              `json:"on_failure"`
}

type field struct {
	Syntax           string
	SemanticElements []string
	Type             string
}

type fieldYml struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description,omitempty"`
	Example     string      `yaml:"example,omitempty"`
	Type        string      `yaml:"type,omitempty"`
	Fields      []*fieldYml `yaml:"fields,omitempty"`
}

// Generate reads a pipeline and creates a fields.yml file based on it.
func Generate(beatsPath, module, fileset string, noDoc bool) error {
	filesetPath := filepath.Join(beatsPath, "module", module, fileset)
	p, err := readPipeline(filesetPath)
	if err != nil {
		return fmt.Errorf("cannot read pipeline: %+v", err)
	}

	data, err := p.toFieldsYml(noDoc)
	if err != nil {
		return fmt.Errorf("cannot generate fields.yml: %+v", err)
	}

	err = writeFieldsYml(filesetPath, data)
	if err != nil {
		return fmt.Errorf("cannot write field.yml: %+v", err)
	}
	return nil
}

func readPipeline(filesetPath string) (*pipeline, error) {
	pipelinePath := filepath.Join(filesetPath, "ingest/pipeline.json")
	r, err := ioutil.ReadFile(pipelinePath)
	if err != nil {
		return nil, err
	}

	var p pipeline
	err = json.Unmarshal(r, &p)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func writeFieldsYml(filesetPath string, fieldsBytes []byte) error {
	output := filepath.Join(filesetPath, "_meta/fields.yml")
	return ioutil.WriteFile(output, fieldsBytes, 0644)
}

func newFieldYml(name, typeName string, noDoc bool) *fieldYml {
	if noDoc {
		return &fieldYml{
			Name: name,
			Type: typeName,
		}
	}

	return &fieldYml{
		Name:        name,
		Type:        typeName,
		Description: "Please add description",
		Example:     "Please add example",
	}
}

func newField(pattern string) field {
	if len(pattern) <= 2 {
		return field{}
	}
	pattern = pattern[1 : len(pattern)-1]

	elements := strings.Split(pattern, ":")
	if !isValidFormat(elements) {
		return field{}
	}

	hint := ""
	if containsType(elements) {
		hint = elements[hintIdx]
	}

	return field{
		Syntax:           elements[typeIdx],
		SemanticElements: strings.Split(elements[elementsIdx], "."),
		Type:             hint,
	}
}

// isValidFormat checks if the input can be split correctly
// 1. if length is 2, the format is {type}:{field.elements}
// 2. if the length is 3, the format is {type}:{field.elements}:{hint}
func isValidFormat(ee []string) bool {
	return len(ee) == 2 || len(ee) == 3
}

// the last element is the type hint
func containsType(ee []string) bool {
	return len(ee) == 3
}

func addNewField(fs []field, f field) []field {
	for _, ff := range fs {
		if reflect.DeepEqual(ff, f) {
			return fs
		}
	}
	return append(fs, f)
}

func getSemanticElementsFromPatterns(patterns []string) ([]field, error) {
	r, err := regexp.Compile("{[\\.\\w\\:]*}")
	if err != nil {
		return nil, err
	}

	var fs []field
	for _, lp := range patterns {
		pp := r.FindAllString(lp, -1)
		for _, p := range pp {
			f := newField(p)
			if f.SemanticElements == nil {
				continue
			}
			fs = addNewField(fs, f)
		}

	}
	return fs, nil
}

func accumulatePatterns(grok interface{}) ([]string, error) {
	for k, v := range grok.(map[string]interface{}) {
		if k == "patterns" {
			vs := v.([]interface{})
			var p []string
			for _, s := range vs {
				p = append(p, s.(string))
			}
			return p, nil
		}
	}
	return nil, fmt.Errorf("No patterns in pipeline")
}

func accumulateRemoveFields(remove interface{}, out []string) []string {
	for k, v := range remove.(map[string]interface{}) {
		if k == "field" {
			switch vs := v.(type) {
			case string:
				return append(out, vs)
			case []string:
				for _, vv := range vs {
					out = append(out, vv)
				}
			case []interface{}:
				for _, vv := range vs {
					vvs := vv.(string)
					out = append(out, vvs)
				}
			default:
				return out

			}
		}
	}
	return out
}

func accumulateRenameFields(rename interface{}, out map[string]string) map[string]string {
	var from, to string
	for k, v := range rename.(map[string]interface{}) {
		if k == "field" {
			from = v.(string)
		}
		if k == "target_field" {
			to = v.(string)
		}
	}
	out[from] = to
	return out
}

type processors struct {
	patterns []string
	remove   []string
	rename   map[string]string
}

func (p *processors) processFields() ([]field, error) {
	f, err := getSemanticElementsFromPatterns(p.patterns)
	if err != nil {
		return nil, err
	}

	for i, ff := range f {
		fs := strings.Join(ff.SemanticElements, ".")
		for k, mv := range p.rename {
			if k == fs {
				ff.SemanticElements = strings.Split(mv, ".")
			}
		}
		for _, rm := range p.remove {
			if fs == rm {
				f = append(f[:i], f[i+1:]...)
			}
		}
	}
	return f, nil
}

func getProcessors(p []map[string]interface{}) (*processors, error) {
	var patterns, rmFields []string
	mvFields := make(map[string]string)

	for _, e := range p {
		if ee, ok := e["grok"]; ok {
			pp, err := accumulatePatterns(ee)
			if err != nil {
				return nil, err
			}
			patterns = append(patterns, pp...)
		}
		if rm, ok := e["remove"]; ok {
			rmFields = accumulateRemoveFields(rm, rmFields)
		}
		if mv, ok := e["rename"]; ok {
			mvFields = accumulateRenameFields(mv, mvFields)
		}
	}

	if patterns == nil {
		return nil, fmt.Errorf("No patterns in pipeline")
	}

	return &processors{
		patterns: patterns,
		remove:   rmFields,
		rename:   mvFields,
	}, nil
}

func getFieldByName(f []*fieldYml, name string) *fieldYml {
	for _, ff := range f {
		if ff.Name == name {
			return ff
		}
	}
	return nil
}

func insertLastField(f []*fieldYml, name string, field field, noDoc bool) []*fieldYml {
	ff := getFieldByName(f, name)
	if ff != nil {
		return f
	}

	fieldType := field.Type
	if fieldType == "" {
		fieldType = types[field.Syntax]
	}
	nf := newFieldYml(name, fieldType, noDoc)
	return append(f, nf)
}

func insertGroup(out []*fieldYml, field field, index, count int, noDoc bool) []*fieldYml {
	g := getFieldByName(out, field.SemanticElements[index])
	if g != nil {
		g.Fields = generateField(g.Fields, field, index+1, count, noDoc)
		return out
	}

	var groupFields []*fieldYml
	groupFields = generateField(groupFields, field, index+1, count, noDoc)
	group := newFieldYml(field.SemanticElements[index], "group", noDoc)
	group.Fields = groupFields
	return append(out, group)
}

func generateField(out []*fieldYml, field field, index, count int, noDoc bool) []*fieldYml {
	if index+1 == count {
		return insertLastField(out, field.SemanticElements[index], field, noDoc)
	}
	return insertGroup(out, field, index, count, noDoc)
}

func generateFields(f []field, noDoc bool) []*fieldYml {
	var out []*fieldYml
	for _, ff := range f {
		index := 1
		if len(ff.SemanticElements) == 1 {
			index = 0
		}
		out = generateField(out, ff, index, len(ff.SemanticElements), noDoc)
	}
	return out
}

func (p *pipeline) toFieldsYml(noDoc bool) ([]byte, error) {
	pr, err := getProcessors(p.Processors)
	if err != nil {
		return nil, err
	}

	var fs []field
	fs, err = pr.processFields()
	if err != nil {
		return nil, err
	}

	f := generateFields(fs, noDoc)
	return yaml.Marshal(&f)
}
