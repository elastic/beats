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

	"github.com/elastic/go-ucfg/yaml"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
	"github.com/elastic/beats/v7/libbeat/mapping"
)

var (
	// Defaults used in the template
	defaultDateDetection           = false
	defaultTotalFieldsLimit        = 10000
	defaultNumberOfRoutingShards   = 30
	defaultMaxDocvalueFieldsSearch = 200

	defaultFields []string
)

// Template holds information for the ES template.
type Template struct {
	sync.Mutex
	name            string
	pattern         string
	elasticLicensed bool
	beatVersion     common.Version
	beatName        string
	esVersion       common.Version
	config          TemplateConfig
	migration       bool
	templateType    IndexTemplateType
	order           int
	priority        int
}

// New creates a new template instance
func New(
	beatVersion string,
	beatName string,
	elasticLicensed bool,
	esVersion common.Version,
	config TemplateConfig,
	migration bool,
) (*Template, error) {
	bV, err := common.NewVersion(beatVersion)
	if err != nil {
		return nil, err
	}

	name := config.Name
	if name == "" {
		name = fmt.Sprintf("%s-%s", beatName, bV.String())
	}

	pattern := config.Pattern
	if pattern == "" {
		pattern = name + "-*"
	}

	event := &beat.Event{
		Fields: common.MapStr{
			// beat object was left in for backward compatibility reason for older configs.
			"beat": common.MapStr{
				"name":    beatName,
				"version": bV.String(),
			},
			"agent": common.MapStr{
				"name":    beatName,
				"version": bV.String(),
			},
			// For the Beats that have an observer role
			"observer": common.MapStr{
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
		templateType:    config.Type,
		order:           config.Order,
		priority:        config.Priority,
	}, nil
}

func (t *Template) load(fields mapping.Fields) (common.MapStr, error) {

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
	properties := common.MapStr{}
	processor := Processor{EsVersion: t.esVersion, ElasticLicensed: t.elasticLicensed, Migration: t.migration}
	if err := processor.Process(fields, nil, properties); err != nil {
		return nil, err
	}

	output := t.Generate(properties, processor.dynamicTemplates)

	return output, nil
}

// LoadFile loads the the template from the given file path
func (t *Template) LoadFile(file string) (common.MapStr, error) {
	fields, err := mapping.LoadFieldsYaml(file)
	if err != nil {
		return nil, err
	}

	return t.load(fields)
}

// LoadBytes loads the template from the given byte array
func (t *Template) LoadBytes(data []byte) (common.MapStr, error) {
	fields, err := loadYamlByte(data)
	if err != nil {
		return nil, err
	}

	return t.load(fields)
}

// LoadMinimal loads the template only with the given configuration
func (t *Template) LoadMinimal() (common.MapStr, error) {
	m := common.MapStr{}
	switch t.templateType {
	case IndexTemplateLegacy:
		m = t.loadMinimalLegacy()
	case IndexTemplateComponent:
		m = t.loadMinimalComponent()
	case IndexTemplateIndex:
		m = t.loadMinimalIndex()
	default:
		return nil, fmt.Errorf("unknown template type %v", t.templateType)
	}

	if t.config.Settings.Source != nil {
		m["mappings"] = buildMappings(
			t.beatVersion, t.esVersion, t.beatName,
			nil, nil,
			common.MapStr(t.config.Settings.Source))
	}

	return m, nil
}

func (t *Template) loadMinimalLegacy() common.MapStr {
	keyPattern, patterns := buildPatternSettings(t.esVersion, t.GetPattern())
	return common.MapStr{
		keyPattern: patterns,
		"order":    t.order,
		"settings": common.MapStr{
			"index": t.config.Settings.Index,
		},
	}
}

func (t *Template) loadMinimalComponent() common.MapStr {
	return common.MapStr{
		"template": common.MapStr{
			"settings": common.MapStr{
				"index": t.config.Settings.Index,
			},
		},
	}
}

func (t *Template) loadMinimalIndex() common.MapStr {
	m := t.loadMinimalComponent()
	m["priority"] = t.priority
	return m
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
func (t *Template) Generate(properties common.MapStr, dynamicTemplates []common.MapStr) common.MapStr {
	switch t.templateType {
	case IndexTemplateLegacy:
		return t.generateLegacy(properties, dynamicTemplates)
	case IndexTemplateComponent:
		return t.generateComponent(properties, dynamicTemplates)
	case IndexTemplateIndex:
		return t.generateIndex(properties, dynamicTemplates)
	}
	return nil
}

func (t *Template) generateLegacy(properties common.MapStr, dynamicTemplates []common.MapStr) common.MapStr {
	keyPattern, patterns := buildPatternSettings(t.esVersion, t.GetPattern())
	return common.MapStr{
		keyPattern: patterns,
		"order":    t.order,
		"mappings": buildMappings(
			t.beatVersion, t.esVersion, t.beatName,
			properties,
			append(dynamicTemplates, buildDynTmpl(t.esVersion)),
			common.MapStr(t.config.Settings.Source)),
		"settings": common.MapStr{
			"index": buildIdxSettings(
				t.esVersion,
				t.config.Settings.Index,
			),
		},
	}
}

func (t *Template) generateComponent(properties common.MapStr, dynamicTemplates []common.MapStr) common.MapStr {
	return common.MapStr{
		"template": common.MapStr{
			"mappings": buildMappings(
				t.beatVersion, t.esVersion, t.beatName,
				properties,
				append(dynamicTemplates, buildDynTmpl(t.esVersion)),
				common.MapStr(t.config.Settings.Source)),
			"settings": common.MapStr{
				"index": buildIdxSettings(
					t.esVersion,
					t.config.Settings.Index,
				),
			},
		},
	}
}

func (t *Template) generateIndex(properties common.MapStr, dynamicTemplates []common.MapStr) common.MapStr {
	tmpl := t.generateComponent(properties, dynamicTemplates)
	tmpl["priority"] = t.priority
	keyPattern, patterns := buildPatternSettings(t.esVersion, t.GetPattern())
	tmpl[keyPattern] = patterns
	return tmpl
}

func buildPatternSettings(ver common.Version, pattern string) (string, interface{}) {
	if ver.Major < 6 {
		return "template", pattern
	}
	return "index_patterns", []string{pattern}
}

func buildMappings(
	beatVersion, esVersion common.Version,
	beatName string,
	properties common.MapStr,
	dynTmpls []common.MapStr,
	source common.MapStr,
) common.MapStr {
	mapping := common.MapStr{
		"_meta": common.MapStr{
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

	major := esVersion.Major
	switch {
	case major == 2:
		mapping.Put("_all.norms.enabled", false)
		mapping = common.MapStr{
			"_default_": mapping,
		}
	case major < 6:
		mapping = common.MapStr{
			"_default_": mapping,
		}
	case major == 6:
		mapping = common.MapStr{
			"doc": mapping,
		}
	case major >= 7:
		// keep typeless structure
	}

	return mapping
}

func buildDynTmpl(ver common.Version) common.MapStr {
	strMapping := common.MapStr{
		"ignore_above": 1024,
		"type":         "keyword",
	}
	if ver.Major == 2 {
		strMapping["type"] = "string"
		strMapping["index"] = "not_analyzed"
	}

	return common.MapStr{
		"strings_as_keyword": common.MapStr{
			"mapping":            strMapping,
			"match_mapping_type": "string",
		},
	}
}

func buildIdxSettings(ver common.Version, userSettings common.MapStr) common.MapStr {
	indexSettings := common.MapStr{
		"refresh_interval": "5s",
		"mapping": common.MapStr{
			"total_fields": common.MapStr{
				"limit": defaultTotalFieldsLimit,
			},
		},
	}

	// number_of_routing shards is only supported for ES version >= 6.1
	// If ES >= 7.0 we can exclude this setting as well.
	version61, _ := common.NewVersion("6.1.0")
	if !ver.LessThan(version61) && ver.Major < 7 {
		indexSettings.Put("number_of_routing_shards", defaultNumberOfRoutingShards)
	}

	if ver.Major >= 7 {
		// copy defaultFields, as defaultFields is shared global slice.
		fields := make([]string, len(defaultFields))
		copy(fields, defaultFields)
		fields = append(fields, "fields.*")

		indexSettings.Put("query.default_field", fields)
	}

	if ver.Major >= 6 {
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
