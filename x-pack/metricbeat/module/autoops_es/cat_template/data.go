// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cat_template

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

const templatePathPrefix = "/_template/"

var (
	templateSchema = s.Schema{
		"order":          c.Int("order", s.Required),
		"version":        c.Int("version", s.Optional),
		"index_patterns": c.Ifc("index_patterns", s.Required), // []string
		"settings":       c.Ifc("settings", s.Required),
		"mappings":       c.Ifc("mappings", s.Required),
		"aliases":        c.Ifc("aliases", s.Required),
	}
)

type CatTemplate struct {
	Name         string `json:"n"`
	ComposedOf   string `json:"c"`
	IndexPattern string `json:"t"`
}

func getNamedTemplates(transactionId string, info *utils.ClusterInfo, templates *map[string]any, reporter t.ReportNamedTemplate) (errs []error) {
	for name, templateData := range *templates {
		if data, ok := templateData.(map[string]any); !ok {
			errs = append(errs, fmt.Errorf("failed casting template data %v", templateData))
		} else if template, err := templateSchema.Apply(data); err != nil {
			errs = append(errs, fmt.Errorf("failed applying template schema for %v: %w", name, err))
		} else {
			template["templateName"] = name
			reporter(transactionId, info, template)
		}
	}

	return errs
}

func eventsMapping(m *elasticsearch.MetricSet, r mb.ReporterV2, info *utils.ClusterInfo, templates *[]CatTemplate) error {
	if len(*templates) == 0 {
		return nil
	}

	partitionedTemplates := t.GetPartitionedTemplates(*templates,
		func(template *CatTemplate) string { return template.Name },
		func(template *CatTemplate) bool {
			return isLegacyTemplate(template) && isNotSystemTemplate(template)
		},
	)

	transactionId, err := t.HandlePartitionedTemplates(m, r, info, templatePathPrefix, partitionedTemplates, getNamedTemplates)
	if err != nil {
		events.SendErrorEvent(err, info, r, CatTemplateMetricSet, CatTemplatePath, transactionId)
		return err
	}

	return nil
}

func isLegacyTemplate(template *CatTemplate) bool {
	return len(template.ComposedOf) == 0
}

func isNotSystemTemplate(template *CatTemplate) bool {
	if len(t.TemplateIndexPatternsToIgnore) == 0 {
		return true
	}
	indexPatterns := utils.ParseArrayOfStrings(template.IndexPattern)

	return len(indexPatterns) == 0 || !utils.AnyMatchesAnyPattern(indexPatterns, t.TemplateIndexPatternsToIgnore)
}
