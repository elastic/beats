// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cat_template

import (
	"slices"
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/templates"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func expectValidParsedDataWithTemplates(templateNames []string) func(t *testing.T, data metricset.FetcherData[[]CatTemplate]) {
	return func(t *testing.T, data metricset.FetcherData[[]CatTemplate]) {
		expectValidParsedDataCheckingTemplateNames(t, data, templateNames)
	}
}

func expectValidParsedDataCheckingTemplateNames(t *testing.T, data metricset.FetcherData[[]CatTemplate], templateNames []string) {
	require.NoError(t, data.Error)
	require.Equal(t, 0, len(data.Reporter.GetErrors()))

	events := data.Reporter.GetEvents()

	require.Equal(t, len(templateNames), len(events))

	auto_ops_testing.CheckAllEventsUseSameTransactionId(t, events)

	for _, event := range events {
		auto_ops_testing.CheckEventWithRandomTransactionId(t, event, data.ClusterInfo)

		// metrics exist
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.order"))
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.index_patterns"))
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.settings"))
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.mappings"))
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.aliases"))
		templateName := auto_ops_testing.GetObjectAsString(t, event.MetricSetFields, "template.templateName")
		require.True(t, slices.Contains(templateNames, templateName), "template '%s' is not in the expected values", templateName)

		// mapper is expected to drop this field if it appears
		require.Nil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.ignored_field"))
	}
}

// Tests that Cluster Info is consistently reported and the Templates are properly reported
func expectValidParsedData(t *testing.T, data metricset.FetcherData[[]CatTemplate]) {
	require.NoError(t, data.Error)
	require.Equal(t, 0, len(data.Reporter.GetErrors()))
	require.LessOrEqual(t, 1, len(data.Reporter.GetEvents()))

	events := data.Reporter.GetEvents()

	auto_ops_testing.CheckAllEventsUseSameTransactionId(t, events)

	for _, event := range events {
		auto_ops_testing.CheckEventWithRandomTransactionId(t, event, data.ClusterInfo)

		// metrics exist
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.order"))
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.index_patterns"))
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.settings"))
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.mappings"))
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.aliases"))

		// mapper is expected to drop this field if it appears
		require.Nil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.ignored_field"))
	}
}

// Tests that Cluster Info is consistently reported and the Templates are properly reported
func expectValidParsedDetailedTemplates(t *testing.T, data metricset.FetcherData[[]CatTemplate]) {
	expectValidParsedData(t, data)

	events := data.Reporter.GetEvents()

	require.Equal(t, 2, len(events))

	event1 := auto_ops_testing.GetEventByName(t, events, "template.templateName", "simple-response")
	event2 := auto_ops_testing.GetEventByName(t, events, "template.templateName", "detailed-response")

	auto_ops_testing.CheckEventWithRandomTransactionId(t, event2, data.ClusterInfo)

	simpleMapping, err := utils.DeserializeData[map[string]interface{}]([]byte(getMappingObject(t, "simple-response")))
	require.NoError(t, err)
	simple, err := templateSchema.Apply(*simpleMapping)
	require.NoError(t, err)

	simpleTemplate := mapstr.M{"template": simple}

	detailedMapping, err := utils.DeserializeData[map[string]interface{}]([]byte(getMappingObject(t, "detailed-response")))
	require.NoError(t, err)
	detailed, err := templateSchema.Apply(*detailedMapping)
	require.NoError(t, err)

	detailedTemplate := mapstr.M{"template": detailed}

	// metrics exist

	// event 1 (simple-response)
	require.Equal(t, "simple-response", auto_ops_testing.GetObjectValue(event1.MetricSetFields, "template.templateName"))
	require.EqualValues(t, 1, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "template.order"))
	require.Nil(t, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "template.version"))
	require.ElementsMatch(t, []string{"*"}, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "template.index_patterns"))
	require.Equal(t, auto_ops_testing.GetObjectAsJson(simpleTemplate, "template.settings"), auto_ops_testing.GetObjectAsJson(event1.MetricSetFields, "template.settings"))
	require.Equal(t, auto_ops_testing.GetObjectAsJson(simpleTemplate, "template.mappings"), auto_ops_testing.GetObjectAsJson(event1.MetricSetFields, "template.mappings"))
	require.Equal(t, auto_ops_testing.GetObjectAsJson(simpleTemplate, "template.aliases"), auto_ops_testing.GetObjectAsJson(event1.MetricSetFields, "template.aliases"))

	// event 2 (detailed-response)
	require.Equal(t, "detailed-response", auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.templateName"))
	require.EqualValues(t, 789, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.order"))
	require.EqualValues(t, 123456, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.version"))
	require.ElementsMatch(t, []string{"a", "b", "c", "d*"}, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.index_patterns"))
	require.Equal(t, auto_ops_testing.GetObjectAsJson(detailedTemplate, "template.settings"), auto_ops_testing.GetObjectAsJson(event2.MetricSetFields, "template.settings"))
	require.Equal(t, auto_ops_testing.GetObjectAsJson(detailedTemplate, "template.mappings"), auto_ops_testing.GetObjectAsJson(event2.MetricSetFields, "template.mappings"))
	require.Equal(t, auto_ops_testing.GetObjectAsJson(detailedTemplate, "template.aliases"), auto_ops_testing.GetObjectAsJson(event2.MetricSetFields, "template.aliases"))

	// schema is expected to drop this field
	require.Nil(t, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "ignored_field"))
	require.Nil(t, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "template.ignored_field"))
	require.Nil(t, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "ignored_field"))
	require.Nil(t, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.ignored_field"))
}

// Tests that Cluster Info is consistently reported and the Templates are properly reported
func expectMixedValidParsedData(t *testing.T, data metricset.FetcherData[[]CatTemplate]) {
	require.ErrorContains(t, data.Error, "fetching templates failed for failed-response")
	require.ErrorContains(t, data.Error, "failed applying template schema for broken-response")

	require.Equal(t, 0, len(data.Reporter.GetErrors()))
	require.Equal(t, 2, len(data.Reporter.GetEvents()))

	events := data.Reporter.GetEvents()

	event := events[0]

	auto_ops_testing.CheckEventWithRandomTransactionId(t, event, data.ClusterInfo)

	// metrics exist
	require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template"))
}

// Expect a valid response from Elasticsearch to create N events
func TestProperlyHandlesResponse(t *testing.T) {
	templates.GivenNoIndexPatternToExclude(t)

	expectedTemplates := []string{".monitoring-es", ".monitoring-beats", "customer-template", ".monitoring-kibana", ".monitoring-logstash", ".monitoring-alerts-7"}

	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cat_template.*.json", auto_ops_testing.SetupSuccessfulTemplateServer(CatTemplatePath, templatePathPrefix, getTemplateResponse), useNamedMetricSet, expectValidParsedDataWithTemplates(expectedTemplates))
}

func TestProperlyHandlesResponseFilteringByName(t *testing.T) {
	templates.GivenSomeIndexNamesToExclude(t, ".monitoring-b*", ".monitoring-a*")
	templates.GivenNoIndexPatternToExclude(t)

	expectedTemplates := []string{".monitoring-es", "customer-template", ".monitoring-kibana", ".monitoring-logstash"}

	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cat_template.*.json", auto_ops_testing.SetupSuccessfulTemplateServer(CatTemplatePath, templatePathPrefix, getTemplateResponse), useNamedMetricSet, expectValidParsedDataWithTemplates(expectedTemplates))
}

// Expect a valid response from Elasticsearch to create N events
func TestProperlyHandlesResponseWithMissingTemplates(t *testing.T) {
	templates.GivenNoIndexPatternToExclude(t)

	expectedTemplates := []string{".monitoring-es", ".monitoring-beats", "customer-template", ".monitoring-kibana", ".monitoring-alerts-7"}

	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cat_template.*.json", auto_ops_testing.SetupSuccessfulTemplateServerWithIgnoredTemplates(CatTemplatePath, templatePathPrefix, getTemplateResponse, []string{".monitoring-logstash"}), useNamedMetricSet, expectValidParsedDataWithTemplates(expectedTemplates))
}

func TestShouldFilterOutSystemTemplates(t *testing.T) {
	expectedTemplates := []string{"customer-template"}

	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cat_template.*.json", auto_ops_testing.SetupSuccessfulTemplateServerWithIgnoredTemplates(CatTemplatePath, templatePathPrefix, getTemplateResponse, []string{".monitoring-logstash"}), useNamedMetricSet, expectValidParsedDataWithTemplates(expectedTemplates))
}

func TestShouldFilterOutWithCustomRule(t *testing.T) {
	templates.GivenSomeIndexPatternsToExclude(t, ".monitoring-beats-7*", ".monitoring-logstash-7*")

	expectedTemplates := []string{".monitoring-es", "customer-template", ".monitoring-kibana", ".monitoring-alerts-7"}

	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cat_template.*.json", auto_ops_testing.SetupSuccessfulTemplateServerWithIgnoredTemplates(CatTemplatePath, templatePathPrefix, getTemplateResponse, []string{".monitoring-logstash"}), useNamedMetricSet, expectValidParsedDataWithTemplates(expectedTemplates))
}

// Expect a valid response from Elasticsearch to create 2 events
func TestProperlyHandlesCustomResponse(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/custom.cat_template.*.json", auto_ops_testing.SetupSuccessfulTemplateServerWithIgnoredTemplates(CatTemplatePath, templatePathPrefix, getTemplateResponse, []string{"ignored-response"}), useNamedMetricSet, expectValidParsedDetailedTemplates)
}

// Expect a valid response from Elasticsearch to create N events
func TestProperlyHandlesInnerErrorsInResponse(t *testing.T) {
	t.Setenv(templates.TEMPLATE_BATCH_SIZE_NAME, "1") // automatically unsets/resets after test

	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/mixed.cat_template.*.json", auto_ops_testing.SetupSuccessfulTemplateServerWithFailedRequests(CatTemplatePath, templatePathPrefix, getTemplateResponse, []string{"failed-response"}), useNamedMetricSet, expectMixedValidParsedData)
}
