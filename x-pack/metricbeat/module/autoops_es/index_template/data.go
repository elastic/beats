// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package index_template

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"
	t "github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/templates"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
)

const templatePathPrefix = "/_index_template/"

var (
	templateSchema = s.Schema{
		// https://www.elastic.co/docs/api/doc/elasticsearch/v8/operation/operation-indices-get-index-template-1
		"index_patterns":                     c.Ifc("index_patterns", s.Required),                     // string | []string
		"composed_of":                        c.Ifc("composed_of", s.Optional),                        // []string
		"template":                           c.Ifc("template", s.Optional),                           // map[string]interface{}
		"version":                            c.Int("version", s.Optional),                            // int64
		"priority":                           c.Int("priority", s.Optional),                           // int64
		"allow_auto_create":                  c.Bool("allow_auto_create", s.Optional),                 // bool
		"data_stream":                        c.Ifc("data_stream", s.Optional),                        // map[string]interface{}
		"deprecated":                         c.Bool("deprecated", s.Optional),                        // bool
		"ignore_missing_component_templates": c.Ifc("ignore_missing_component_templates", s.Optional), // string | []string
		"_meta":                              c.Ifc("_meta", s.Optional),                              // map[string]interface{}
	}
)

type IndexTemplate struct {
	Name          string         `json:"name"`
	IndexTemplate map[string]any `json:"index_template"`
}

type IndexTemplates struct {
	Templates []IndexTemplate `json:"index_templates"`
}

func getNamedTemplates(transactionId string, info *utils.ClusterInfo, templates *IndexTemplates, reporter t.ReportNamedTemplate) (errs []error) {
	for _, templateData := range templates.Templates {
		template, err := templateSchema.Apply(templateData.IndexTemplate)

		if err != nil {
			errs = append(errs, fmt.Errorf("failed applying index template schema for %v: %w", templateData.Name, err))
			continue
		}

		template["templateName"] = templateData.Name

		reporter(transactionId, info, template)
	}

	return errs
}

func eventsMapping(m *elasticsearch.MetricSet, r mb.ReporterV2, info *utils.ClusterInfo, response *IndexTemplates) error {
	if len(response.Templates) == 0 {
		return nil
	}

	partitionedTemplates, errors := t.GetPartitionedTemplatesWithErrors(response.Templates,
		extractTemplateName,
		isNotSystemTemplate)

	for _, err := range errors {
		m.Logger().Warn("Failed to extract index templates: %v", err)
	}

	err := t.HandleIndividualTemplateRequests(m, r, info, templatePathPrefix, partitionedTemplates, getNamedTemplates)

	if err != nil {
		err = fmt.Errorf("failed applying index template schema: %w", err)
		events.SendErrorEventWithRandomTransactionId(err, info, r, IndexTemplateMetricSet, IndexTemplatePath)
		return err
	}

	return nil
}

// Extracts the name of the IndexTemplate.
func extractTemplateName(template *IndexTemplate) string {
	return template.Name
}

func isNotSystemTemplate(template *IndexTemplate) (bool, error) {
	isSystemTemplate, err := isSystemTemplate(template)
	if err != nil {
		return false, err
	}
	return !isSystemTemplate, nil
}

func isSystemTemplate(template *IndexTemplate) (bool, error) {
	if template.IndexTemplate == nil {
		return false, fmt.Errorf("index_template is nil")
	}

	if template.IndexTemplate["_meta"] != nil {
		meta, ok := template.IndexTemplate["_meta"].(map[string]any)

		if ok && meta["managed"] != nil {
			if managed, ok := meta["managed"].(bool); ok {
				return managed, nil
			}
		}
	}

	if len(t.TemplateIndexPatternsToIgnore) == 0 {
		return false, nil
	}

	indexPatterns, err := template.getIndexPatterns()
	if err != nil {
		return false, err
	}

	return len(indexPatterns) == 0 || utils.AnyMatchesAnyPattern(indexPatterns, t.TemplateIndexPatternsToIgnore), nil
}

func (template *IndexTemplate) getIndexPatterns() ([]string, error) {
	rawValues, ok := template.IndexTemplate["index_patterns"]
	if !ok {
		return nil, fmt.Errorf("index_patterns not found in template %q", template.Name)
	}

	rawSlice, ok := rawValues.([]any)
	if !ok {
		return nil, fmt.Errorf("index_patterns in template %q is not a slice", template.Name)
	}

	indexPatterns := make([]string, 0, len(rawSlice))
	for _, v := range rawSlice {
		str, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("index_patterns in template %q contains a non-string element", template.Name)
		}
		indexPatterns = append(indexPatterns, str)
	}
	return indexPatterns, nil
}
