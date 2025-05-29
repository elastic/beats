// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package component_template

import (
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"
	t "github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/templates"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
)

var (
	templatePathPrefix = "/_component_template/"
	templateSchema     = s.Schema{
		// https://www.elastic.co/docs/api/doc/elasticsearch/v8/operation/operation-cluster-get-component-template-1
		"template": c.Ifc("template", s.Required), // map[string]interface{}
		"version":  c.Int("version", s.Optional),  // int64
		"_meta":    c.Ifc("_meta", s.Optional),    // map[string]interface{}
	}
)

type ComponentTemplate struct {
	Name              string                 `json:"name"`
	ComponentTemplate map[string]interface{} `json:"component_template"`
}

type ComponentTemplates struct {
	Templates []ComponentTemplate `json:"component_templates"`
}

func getNamedTemplates(transactionId string, info *utils.ClusterInfo, templates *ComponentTemplates, reporter t.ReportNamedTemplate) (errs []error) {
	for _, templateData := range templates.Templates {
		template, err := templateSchema.Apply(templateData.ComponentTemplate)

		if err != nil {
			errs = append(errs, fmt.Errorf("failed applying component template schema for %v: %w", templateData.Name, err))
			continue
		}

		template["templateName"] = templateData.Name

		reporter(transactionId, info, template)
	}

	return errs
}

func eventsMapping(m *elasticsearch.MetricSet, r mb.ReporterV2, info *utils.ClusterInfo, response *ComponentTemplates) error {
	if len(response.Templates) == 0 {
		return nil
	}

	partitionedTemplates := t.GetPartitionedTemplates(response.Templates,
		extractTemplateName,
		isTemplateNotManaged)

	err := t.HandleIndividualTemplateRequests(m, r, info, templatePathPrefix, partitionedTemplates, getNamedTemplates)

	if err != nil {
		err = fmt.Errorf("failed applying component template schema %w", err)
		events.SendErrorEventWithRandomTransactionId(err, info, r, ComponentTemplateMetricSet, ComponentTemplatePath)
		return err
	}

	return nil
}

// Extracts the name of the ComponentTemplate.
func extractTemplateName(template *ComponentTemplate) string {
	return template.Name
}

// Determines if the ComponentTemplate is not managed.
func isTemplateNotManaged(template *ComponentTemplate) bool {
	managed := false

	if template.ComponentTemplate != nil && template.ComponentTemplate["_meta"] != nil {
		meta := template.ComponentTemplate["_meta"].(map[string]interface{})

		managed = meta["managed"] != nil && meta["managed"].(bool)
	}

	return !managed
}
