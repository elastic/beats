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

package template

import (
	"fmt"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/version"
	"github.com/elastic/go-ucfg/yaml"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
	"github.com/elastic/beats/v7/libbeat/mapping"
)

var (
	// Defaults used in the template
	defaultDateDetection           = false
	defaultTotalFieldsLimit        = 10000
	defaultMaxDocvalueFieldsSearch = 200

	defaultFields []string
)

// Template holds information for the ES template.
type Template struct {
	sync.Mutex
	name            string
	pattern         string
	elasticLicensed bool
	beatVersion     version.V
	beatName        string
	esVersion       version.V
	config          TemplateConfig
	migration       bool
	priority        int
	isServerless    bool
}

// New creates a new template instance
func New(
	isServerless bool,
	beatVersion string,
	beatName string,
	elasticLicensed bool,
	esVersion version.V,
	config TemplateConfig,
	migration bool,
) (*Template, error) {
	bV, err := version.New(beatVersion)
	if err != nil {
		return nil, err
	}

	name := config.Name
	if config.JSON.Enabled {
		name = config.JSON.Name
	}

	if name == "" {
		name = fmt.Sprintf("%s-%s", beatName, bV.String())
	}

	pattern := config.Pattern
	if pattern == "" {
		pattern = name + "*"
	}

	event := &beat.Event{
		Fields: mapstr.M{
			// beat object was left in for backward compatibility reason for older configs.
			"beat": mapstr.M{
				"name":    beatName,
				"version": bV.String(),
			},
			"agent": mapstr.M{
				"name":    beatName,
				"version": bV.String(),
			},
			// For the Beats that have an observer role
			"observer": mapstr.M{
				"name":    beatName,
				"version": bV.String(),
			},
		},
		Timestamp: time.Now(),
	}

	nameFormatter, err := fmtstr.CompileEvent(name)
	if err != nil {
		return nil, err
	}
	name, err = nameFormatter.Run(event)
	if err != nil {
		return nil, err
	}

	patternFormatter, err := fmtstr.CompileEvent(pattern)
	if err != nil {
		return nil, err
	}
	pattern, err = patternFormatter.Run(event)
	if err != nil {
		return nil, err
	}

	// In case no esVersion is set, it is assumed the same as beat version
	if !esVersion.IsValid() {
		esVersion = *bV
	}

	return &Template{
		pattern:         pattern,
		name:            name,
		elasticLicensed: elasticLicensed,
		beatVersion:     *bV,
		esVersion:       esVersion,
		beatName:        beatName,
		config:          config,
		migration:       migration,
		priority:        config.Priority,
		isServerless:    isServerless,
	}, nil
}

func (t *Template) load(fields mapping.Fields) (mapstr.M, error) {

	// Locking to make sure dynamicTemplates and defaultFields is not accessed in parallel
	t.Lock()
	defer t.Unlock()

	defaultFields = nil

	var err error
	if len(t.config.AppendFields) > 0 {
		fields, err = mapping.ConcatFields(fields, t.config.AppendFields)
		if err != nil {
			return nil, err
		}
	}

	// Start processing at the root
	properties := mapstr.M{}
	analyzers := mapstr.M{}
	processor := Processor{EsVersion: t.esVersion, ElasticLicensed: t.elasticLicensed, Migration: t.migration}
	if err := processor.Process(fields, nil, properties, analyzers); err != nil {
		return nil, err
	}

	output := t.Generate(properties, analyzers, processor.dynamicTemplates)

	return output, nil
}

// LoadFile loads the the template from the given file path
func (t *Template) LoadFile(file string) (mapstr.M, error) {
	fields, err := mapping.LoadFieldsYaml(file)
	if err != nil {
		return nil, err
	}

	return t.load(fields)
}

// LoadBytes loads the template from the given byte array
func (t *Template) LoadBytes(data []byte) (mapstr.M, error) {
	fields, err := loadYamlByte(data)
	if err != nil {
		return nil, err
	}

	return t.load(fields)
}

// LoadMinimal loads the template only with the given configuration
func (t *Template) LoadMinimal() mapstr.M {
	templ := mapstr.M{}
	if t.config.Settings.Source != nil {
		templ["mappings"] = buildMappings(
			t.beatVersion, t.beatName,
			nil, nil,
			mapstr.M(t.config.Settings.Source))
	}
	// delete default settings not available on serverless
	if _, ok := t.config.Settings.Index["number_of_shards"]; ok && t.isServerless {
		delete(t.config.Settings.Index, "number_of_shards")
	}
	templ["settings"] = mapstr.M{
		"index": t.config.Settings.Index,
	}
	return mapstr.M{
		"template":       templ,
		"data_stream":    struct{}{},
		"priority":       t.priority,
		"index_patterns": []string{t.GetPattern()},
	}
}

// GetName returns the name of the template
func (t *Template) GetName() string {
	return t.name
}

// GetPattern returns the pattern of the template
func (t *Template) GetPattern() string {
	return t.pattern
}

// Generate generates the full template
// The default values are taken from the default variable.
func (t *Template) Generate(properties, analyzers mapstr.M, dynamicTemplates []mapstr.M) mapstr.M {
	tmpl := t.generateComponent(properties, analyzers, dynamicTemplates)
	tmpl["data_stream"] = struct{}{}
	tmpl["priority"] = t.priority
	tmpl["index_patterns"] = []string{t.GetPattern()}
	return tmpl

}

func (t *Template) generateComponent(properties, analyzers mapstr.M, dynamicTemplates []mapstr.M) mapstr.M {
	m := mapstr.M{
		"template": mapstr.M{
			"mappings": buildMappings(
				t.beatVersion, t.beatName,
				properties,
				append(dynamicTemplates, buildDynTmpl(t.esVersion)),
				mapstr.M(t.config.Settings.Source)),
			"settings": mapstr.M{
				"index": buildIdxSettings(
					t.esVersion,
					t.config.Settings.Index,
					t.isServerless,
				),
			},
		},
	}
	if len(t.config.Settings.Lifecycle) > 0 {
		m.Put("template.lifecycle", t.config.Settings.Lifecycle)
	}
	if len(analyzers) != 0 {
		m.Put("template.settings.analysis.analyzer", analyzers)
	}
	return m
}

func buildMappings(
	beatVersion version.V,
	beatName string,
	properties mapstr.M,
	dynTmpls []mapstr.M,
	source mapstr.M,
) mapstr.M {
	mapping := mapstr.M{
		"_meta": mapstr.M{
			"version": beatVersion.String(),
			"beat":    beatName,
		},
		"date_detection":    defaultDateDetection,
		"dynamic_templates": dynTmpls,
		"properties":        properties,
	}

	if len(source) > 0 {
		mapping["_source"] = source
	}

	return mapping
}

func buildDynTmpl(ver version.V) mapstr.M {
	return mapstr.M{
		"strings_as_keyword": mapstr.M{
			"mapping": mapstr.M{
				"ignore_above": 1024,
				"type":         "keyword",
			},
			"match_mapping_type": "string",
		},
	}
}

func buildIdxSettings(ver version.V, userSettings mapstr.M, isServerless bool) mapstr.M {
	indexSettings := mapstr.M{
		"refresh_interval": "5s",
		"mapping": mapstr.M{
			"total_fields": mapstr.M{
				"limit": defaultTotalFieldsLimit,
			},
		},
	}

	// copy defaultFields, as defaultFields is shared global slice.
	fields := make([]string, len(defaultFields))
	copy(fields, defaultFields)
	fields = append(fields, "fields.*")

	indexSettings.Put("query.default_field", fields)

	// deal with settings that aren't available on serverless
	if isServerless {
		logp.L().Infof("remote instance is serverless, number_of_shards and max_docvalue_fields_search will be skipped in index template")
		userSettings.Delete("number_of_shards")
	} else {
		indexSettings.Put("max_docvalue_fields_search", defaultMaxDocvalueFieldsSearch)
	}

	indexSettings.DeepUpdate(userSettings)
	return indexSettings
}

func loadYamlByte(data []byte) (mapping.Fields, error) {
	cfg, err := yaml.NewConfig(data)
	if err != nil {
		return nil, err
	}

	var keys []mapping.Field
	err = cfg.Unpack(&keys)
	if err != nil {
		return nil, err
	}

	fields := mapping.Fields{}
	for _, key := range keys {
		fields = append(fields, key.Fields...)
	}
	return fields, nil
}
