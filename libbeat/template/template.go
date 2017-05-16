package template

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/go-ucfg/yaml"
)

var (
	// Defaults used in the template
	defaultDateDetection    = false
	defaultTotalFieldsLimit = 10000

	// Array to store dynamicTemplate parts in
	dynamicTemplates []common.MapStr
)

type Template struct {
	index       string
	beatVersion Version
	esVersion   Version
	settings    templateSettings
}

// New creates a new template instance
func New(beatVersion string, esVersion string, index string, settings templateSettings) (*Template, error) {

	bV, err := NewVersion(beatVersion)
	if err != nil {
		return nil, err
	}

	// In case no esVersion is set, it is assumed the same as beat version
	if esVersion == "" {
		esVersion = beatVersion
	}

	esV, err := NewVersion(esVersion)
	if err != nil {
		return nil, err
	}

	return &Template{
		index:       index,
		beatVersion: *bV,
		esVersion:   *esV,
		settings:    settings,
	}, nil

}

// Load the given input and generates the input based on it
func (t *Template) Load(file string) (common.MapStr, error) {

	fields, err := loadYaml(file)
	if err != nil {
		return nil, err
	}

	// Start processing at the root
	properties := fields.process("", t.esVersion)
	output := t.generate(properties, dynamicTemplates)

	return output, nil
}

// GetName returns the name of the template which is {index}-{version}
func (t *Template) GetName() string {
	return fmt.Sprintf("%s-%s", t.index, t.beatVersion.String())
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
	indexSettings.DeepUpdate(t.settings.Index)

	// Load basic structure
	basicStructure := common.MapStr{
		"mappings": common.MapStr{
			"_default_": common.MapStr{
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

	if len(t.settings.Source) > 0 {
		basicStructure.Put("mappings._default_._source", t.settings.Source)
	}

	// ES 6 moved from template to index_patterns: https://github.com/elastic/elasticsearch/pull/21009
	if t.esVersion.major >= 6 {
		basicStructure.Put("index_patterns", []string{t.GetName() + "-*"})
	} else {
		basicStructure.Put("template", t.GetName()+"-*")
	}

	if t.esVersion.IsMajor(2) || t.esVersion.IsMajor(5) {
		basicStructure.Put("mappings._default_._all.norms.enabled", false)
	}

	return basicStructure
}

func loadYaml(path string) (Fields, error) {
	keys := []Field{}

	cfg, err := yaml.NewConfigWithFile(path)
	if err != nil {
		return nil, err
	}
	cfg.Unpack(&keys)

	fields := Fields{}

	for _, key := range keys {
		fields = append(fields, key.Fields...)
	}
	return fields, nil
}
