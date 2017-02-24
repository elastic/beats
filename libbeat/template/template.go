package template

import (
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
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

// GetTemplate creates a template based on the given inputs
func GetTemplate(version string, beatName string, files []string) (common.MapStr, error) {

	beatVersion := beat.GetDefaultVersion()

	// In case no esVersion is set, it is assumed the same as beat version
	if version == "" {
		version = beatVersion
	}

	esVersion, err := NewVersion(version)
	if err != nil {
		return nil, err
	}

	fields := Fields{}

	for _, file := range files {
		f, err := loadYaml(file)
		if err != nil {
			return nil, err
		}
		fields = append(fields, f...)
	}

	// Start processing at the root
	properties := fields.process("", *esVersion)

	indexPattern := fmt.Sprintf("%s-%s-*", beatName, beatVersion)
	output := createTemplate(properties, beatVersion, *esVersion, indexPattern, dynamicTemplates)

	return output, nil

}

// createTemplate creates the full template
// The default values are taken from the default variable.
func createTemplate(properties common.MapStr, version string, esVersion Version, indexPattern string, dynamicTemplates []common.MapStr) common.MapStr {

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

	if esVersion.IsMajor(2) {
		dynamicTemplateBase.Put("strings_as_keyword.mapping.type", "string")
		dynamicTemplateBase.Put("strings_as_keyword.mapping.index", "not_analyzed")
	}

	dynamicTemplates = append(dynamicTemplates, dynamicTemplateBase)

	// Load basic structure
	basicStructure := common.MapStr{
		"mappings": common.MapStr{
			"_default_": common.MapStr{
				"_meta": common.MapStr{
					"version": version,
				},
				"date_detection":    defaultDateDetection,
				"dynamic_templates": dynamicTemplates,
				"properties":        properties,
			},
		},
		"order": 0,
		"settings": common.MapStr{
			"index.refresh_interval": "5s",
		},
		"template": indexPattern,
	}

	if esVersion.IsMajor(2) {
		basicStructure.Put("mappings._default_._all.norms.enabled", false)
	} else {
		// Metricbeat exceeds the default of 1000 fields
		basicStructure.Put("settings.index.mapping.total_fields.limit", defaultTotalFieldsLimit)
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
