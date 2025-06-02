// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package component_template

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/templates"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Tests that Cluster Info is consistently reported and the Templates are properly reported
func expectValidParsedData(t *testing.T, data metricset.FetcherData[ComponentTemplates]) {
	require.NoError(t, data.Error)
	require.Equal(t, 0, len(data.Reporter.GetErrors()))
	require.LessOrEqual(t, 1, len(data.Reporter.GetEvents()))

	events := data.Reporter.GetEvents()

	auto_ops_testing.CheckAllEventsUseSameTransactionId(t, events)

	for _, event := range events {
		auto_ops_testing.CheckEventWithRandomTransactionId(t, event, data.ClusterInfo)

		// metrics exist
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.template"))

		// mapper is expected to drop this field if it appears
		require.Nil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.ignored_field"))
	}
}

func expectValidParsedDataWithTemplates(templateNames []string) func(t *testing.T, data metricset.FetcherData[ComponentTemplates]) {
	return func(t *testing.T, data metricset.FetcherData[ComponentTemplates]) {
		expectValidParsedDataWithTemplateNames(t, data, templateNames)
	}
}

// Tests that Cluster Info is consistently reported and the Templates are properly reported
func expectValidParsedDataWithTemplateNames(t *testing.T, data metricset.FetcherData[ComponentTemplates], templateNames []string) {
	require.NoError(t, data.Error)
	require.Equal(t, 0, len(data.Reporter.GetErrors()))

	events := data.Reporter.GetEvents()

	var actualTemplateNames []string
	for _, event := range events {
		name := auto_ops_testing.GetObjectAsString(t, event.MetricSetFields, "template.templateName")
		actualTemplateNames = append(actualTemplateNames, name)
	}
	require.Equal(t, len(templateNames), len(actualTemplateNames), "Wrong number of template names %s", actualTemplateNames)

	auto_ops_testing.CheckAllEventsUseSameTransactionId(t, events)

	for _, event := range events {
		auto_ops_testing.CheckEventWithRandomTransactionId(t, event, data.ClusterInfo)

		// metrics exist
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.template"))

		// mapper is expected to drop this field if it appears
		require.Nil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.ignored_field"))
	}
}

// Tests that Cluster Info is consistently reported and the Templates are properly reported
func expectValidParsedDetailedTemplates(t *testing.T, data metricset.FetcherData[ComponentTemplates]) {
	expectValidParsedData(t, data)

	events := data.Reporter.GetEvents()

	require.Equal(t, 2, len(events))

	event1 := auto_ops_testing.GetEventByName(t, events, "template.templateName", "simple-response")
	event2 := auto_ops_testing.GetEventByName(t, events, "template.templateName", "detailed-response")

	auto_ops_testing.CheckEventWithRandomTransactionId(t, event2, data.ClusterInfo)

	simpleMapping, err := utils.DeserializeData[map[string]interface{}]([]byte(getMappingObject(t, "simple-response")))
	require.NoError(t, err)
	simple, err := templateSchema.Apply((*simpleMapping)["component_template"].(map[string]interface{}))
	require.NoError(t, err)

	simpleTemplate := mapstr.M{"template": simple}

	detailedMapping, err := utils.DeserializeData[map[string]interface{}]([]byte(getMappingObject(t, "detailed-response")))
	require.NoError(t, err)
	detailed, err := templateSchema.Apply((*detailedMapping)["component_template"].(map[string]interface{}))
	require.NoError(t, err)

	detailedTemplate := mapstr.M{"template": detailed}

	// metrics exist

	// event 1 (simple-response)
	require.Equal(t, "simple-response", auto_ops_testing.GetObjectValue(event1.MetricSetFields, "template.templateName"))
	require.Nil(t, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "template.version"))
	require.Equal(t, auto_ops_testing.GetObjectAsJson(simpleTemplate, "template.template.settings"), auto_ops_testing.GetObjectAsJson(event1.MetricSetFields, "template.template.settings"))
	require.Nil(t, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "template._meta"))

	// event 2 (detailed-response)
	require.Equal(t, "detailed-response", auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.templateName"))
	require.EqualValues(t, 123456, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.version"))
	require.Equal(t, auto_ops_testing.GetObjectAsJson(detailedTemplate, "template.template"), auto_ops_testing.GetObjectAsJson(event2.MetricSetFields, "template.template"))
	require.EqualValues(t, 456, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template._meta.property1"))

	// schema is expected to drop this field
	require.Nil(t, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "ignored_field"))
	require.Nil(t, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "template.ignored_field"))
	require.Nil(t, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "ignored_field"))
	require.Nil(t, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.ignored_field"))
}

// Tests that Cluster Info is consistently reported and the Templates are properly reported
func expectMixedValidParsedData(t *testing.T, data metricset.FetcherData[ComponentTemplates]) {
	require.ErrorContains(t, data.Error, "fetching templates failed for failed-response")
	require.ErrorContains(t, data.Error, "failed applying component template schema for broken-response")

	require.Equal(t, 0, len(data.Reporter.GetErrors()))
	require.Equal(t, 2, len(data.Reporter.GetEvents()))

	events := data.Reporter.GetEvents()

	event := events[0]

	auto_ops_testing.CheckEventWithRandomTransactionId(t, event, data.ClusterInfo)

	// metrics exist
	require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template"))
}

// Expect a valid response from Elasticsearch to create N events
func TestProperlyHandlesResponseV7(t *testing.T) {
	expectedTemplateNames := []string{
		".alerts-ecs-mappings",
		"my-component-template-1",
		"my-component-template-2",
		".alerts-technical-mappings",
		".alerts-observability.apm.alerts-mappings",
		".alerts-observability.logs.alerts-mappings",
		".alerts-observability.uptime.alerts-mappings",
		".alerts-observability.metrics.alerts-mappings",
	}
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/component_template.7*.json", auto_ops_testing.SetupSuccessfulTemplateServer(ComponentTemplatePath, templatePathPrefix, getTemplateResponse), useNamedMetricSet, expectValidParsedDataWithTemplates(expectedTemplateNames))
}

func TestProperlyHandlesResponseV8(t *testing.T) {
	expectedTemplateNames := []string{
		"entities_v1_event",
		"entities_v1_entity",
		"kibana-reporting@custom",
		"entities_v1_latest_base",
		"my-component-template-1",
		"my-component-template-2",
		"entities_v1_history_base",
		".alerts-technical-mappings",
		".preview.alerts-security.alerts-mappings",
		".kibana-observability-ai-assistant-component-template-kb",
		".kibana-observability-ai-assistant-component-template-conversations",
	}
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/component_template.8*.json", auto_ops_testing.SetupSuccessfulTemplateServer(ComponentTemplatePath, templatePathPrefix, getTemplateResponse), useNamedMetricSet, expectValidParsedDataWithTemplates(expectedTemplateNames))
}

// Expect a valid response from Elasticsearch to create N events
func TestProperlyHandlesResponseWithMissingTemplates(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/component_template.*.json", auto_ops_testing.SetupSuccessfulTemplateServerWithIgnoredTemplates(ComponentTemplatePath, templatePathPrefix, getTemplateResponse, []string{"my-component-template-1"}), useNamedMetricSet, expectValidParsedData)
}

// Expect a valid response from Elasticsearch to create 2 events
func TestProperlyHandlesCustomResponse(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/custom.component_template.*.json", auto_ops_testing.SetupSuccessfulTemplateServerWithIgnoredTemplates(ComponentTemplatePath, templatePathPrefix, getTemplateResponse, []string{"ignored-response"}), useNamedMetricSet, expectValidParsedDetailedTemplates)
}

// Expect a valid response from Elasticsearch to create N events
func TestProperlyHandlesInnerErrorsInResponse(t *testing.T) {
	t.Setenv(templates.TEMPLATE_BATCH_SIZE_NAME, "1") // automatically unsets/resets after test

	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/mixed.component_template.*.json", auto_ops_testing.SetupSuccessfulTemplateServerWithFailedRequests(ComponentTemplatePath, templatePathPrefix, getTemplateResponse, []string{"failed-response"}), useNamedMetricSet, expectMixedValidParsedData)
}
