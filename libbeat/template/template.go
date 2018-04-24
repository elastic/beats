package template

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/common/fmtstr"
)

var (
	// Defaults used in the template
	defaultDateDetection         = false
	defaultTotalFieldsLimit      = 10000
	defaultNumberOfRoutingShards = 30

	// Array to store dynamicTemplate parts in
	dynamicTemplates []common.MapStr
)

type Template struct {
	name        string
	pattern     string
	beatVersion common.Version
	esVersion   common.Version
	config      TemplateConfig
}

// New creates a new template instance
func New(beatVersion string, beatName string, esVersion string, config TemplateConfig) (*Template, error) {
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
			"beat": common.MapStr{
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
	if esVersion == "" {
		esVersion = beatVersion
	}

	esV, err := common.NewVersion(esVersion)
	if err != nil {
		return nil, err
	}

	return &Template{
		pattern:     pattern,
		name:        name,
		beatVersion: *bV,
		esVersion:   *esV,
		config:      config,
	}, nil
}

// Load the given input and generates the input based on it
func (t *Template) Load(file string) (common.MapStr, error) {

	fields, err := common.LoadFieldsYaml(file)
	if err != nil {
		return nil, err
	}

	if len(t.config.AppendFields) > 0 {
		cfgwarn.Experimental("append_fields is used.")
		fields, err = appendFields(fields, t.config.AppendFields)
		if err != nil {
			return nil, err
		}
	}

	// Start processing at the root
	properties := common.MapStr{}
	processor := Processor{EsVersion: t.esVersion}
	if err := processor.process(fields, "", properties); err != nil {
		return nil, err
	}
	output := t.generate(properties, dynamicTemplates)

	return output, nil
}

// GetName returns the name of the template
func (t *Template) GetName() string {
	return t.name
}

// GetPattern returns the pattern of the template
func (t *Template) GetPattern() string {
	return t.pattern
}

// generate generates the full template
// The default values are taken from the default variable.
func (t *Template) generate(properties common.MapStr, dynamicTemplates []common.MapStr) common.MapStr {
	// Add base dynamic template
	var dynamicTemplateBase = common.MapStr{
		"strings_as_keyword": common.MapStr{
			"mapping": common.MapStr{
				"ignore_above": 1024,
				"type":         "keyword",
			},
			"match_mapping_type": "string",
		},
	}

	if t.esVersion.IsMajor(2) {
		dynamicTemplateBase.Put("strings_as_keyword.mapping.type", "string")
		dynamicTemplateBase.Put("strings_as_keyword.mapping.index", "not_analyzed")
	}

	dynamicTemplates = append(dynamicTemplates, dynamicTemplateBase)

	indexSettings := common.MapStr{
		"refresh_interval": "5s",
		"mapping": common.MapStr{
			"total_fields": common.MapStr{
				"limit": defaultTotalFieldsLimit,
			},
		},
	}

	// number_of_routing shards is only supported for ES version >= 6.1
	version61, _ := common.NewVersion("6.1.0")
	if !t.esVersion.LessThan(version61) {
		indexSettings.Put("number_of_routing_shards", defaultNumberOfRoutingShards)
	}

	indexSettings.DeepUpdate(t.config.Settings.Index)

	var mappingName string
	if t.esVersion.Major >= 6 {
		mappingName = "doc"
	} else {
		mappingName = "_default_"
	}

	// Load basic structure
	basicStructure := common.MapStr{
		"mappings": common.MapStr{
			mappingName: common.MapStr{
				"_meta": common.MapStr{
					"version": t.beatVersion.String(),
				},
				"date_detection":    defaultDateDetection,
				"dynamic_templates": dynamicTemplates,
				"properties":        properties,
			},
		},
		"order": 1,
		"settings": common.MapStr{
			"index": indexSettings,
		},
	}

	if len(t.config.Settings.Source) > 0 {
		key := fmt.Sprintf("mappings.%s._source", mappingName)
		basicStructure.Put(key, t.config.Settings.Source)
	}

	// ES 6 moved from template to index_patterns: https://github.com/elastic/elasticsearch/pull/21009
	if t.esVersion.Major >= 6 {
		basicStructure.Put("index_patterns", []string{t.GetPattern()})
	} else {
		basicStructure.Put("template", t.GetPattern())
	}

	if t.esVersion.IsMajor(2) {
		basicStructure.Put("mappings._default_._all.norms.enabled", false)
	}

	return basicStructure
}

func appendFields(fields, appendFields common.Fields) (common.Fields, error) {
	if len(appendFields) > 0 {
		appendFieldKeys := appendFields.GetKeys()

		// Append is only allowed to add fields, not overwrite
		for _, key := range appendFieldKeys {
			if fields.HasNode(key) {
				return nil, fmt.Errorf("append_fields contains an already existing key: %s", key)
			}
		}
		// Appends fields to existing fields
		fields = append(fields, appendFields...)
	}
	return fields, nil
}
