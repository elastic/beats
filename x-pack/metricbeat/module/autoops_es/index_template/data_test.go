// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package index_template

import (
	"reflect"
	"slices"
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/templates"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Tests that Cluster Info is consistently reported and the Templates are properly reported
func expectValidParsedData(t *testing.T, data metricset.FetcherData[IndexTemplates]) {
	expectValidParsedDataSkippingComposedOfCheck(t, data, map[string]struct{}{})
}

func expectValidParsedDataSkippingComposedOfCheck(t *testing.T, data metricset.FetcherData[IndexTemplates], skipComposedOfCheckFor map[string]struct{}) {
	require.NoError(t, data.Error)
	require.Equal(t, 0, len(data.Reporter.GetErrors()))
	require.LessOrEqual(t, 1, len(data.Reporter.GetEvents()))

	events := data.Reporter.GetEvents()

	auto_ops_testing.CheckAllEventsUseSameTransactionId(t, events)

	for _, event := range events {
		auto_ops_testing.CheckEventWithRandomTransactionId(t, event, data.ClusterInfo)

		// metrics exist
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.priority"))
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.index_patterns"))
		templateName := auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.templateName")
		if nameStr, ok := templateName.(string); ok {
			if _, skip := skipComposedOfCheckFor[nameStr]; !skip {
				require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.composed_of"))
			}
		} else {
			t.Logf("Unable to retrieve template.name (not a string): %v", templateName)
		}
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.template"))

		// mapper is expected to drop this field if it appears
		require.Nil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.ignored_field"))
	}
}

func expectValidParsedDataWithTemplates(templateNames []string) func(t *testing.T, data metricset.FetcherData[IndexTemplates]) {
	return func(t *testing.T, data metricset.FetcherData[IndexTemplates]) {
		expectValidParsedDataWithTemplateNames(t, data, templateNames)
	}
}

// Tests that Cluster Info is consistently reported and the Templates are properly reported
func expectValidParsedDataWithTemplateNames(t *testing.T, data metricset.FetcherData[IndexTemplates], templateNames []string) {
	require.NoError(t, data.Error)
	require.Equal(t, 0, len(data.Reporter.GetErrors()))
	require.LessOrEqual(t, 1, len(data.Reporter.GetEvents()))

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
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.priority"))
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.index_patterns"))
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.composed_of"))
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.template"))
		templateName := auto_ops_testing.GetObjectAsString(t, event.MetricSetFields, "template.templateName")
		require.True(t, slices.Contains(templateNames, templateName), "template '%s' is not in the expected values", templateName)

		// mapper is expected to drop this field if it appears
		require.Nil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "template.ignored_field"))
	}
}

// Tests that Cluster Info is consistently reported and the Templates are properly reported
func expectValidParsedDetailedTemplates(t *testing.T, data metricset.FetcherData[IndexTemplates]) {
	expectValidParsedData(t, data)

	expectValidParsedDetailedTemplatesCommon(t, data)

	expectValidParsedDetailedTemplatesOptional(t, data)
}

func expectValidParsedDetailedTemplatesWithoutComposedOf(t *testing.T, data metricset.FetcherData[IndexTemplates]) {
	skipComposedOfCheckFor := map[string]struct{}{
		"detailed-response": {},
	}

	expectValidParsedDataSkippingComposedOfCheck(t, data, skipComposedOfCheckFor)

	expectValidParsedDetailedTemplatesCommon(t, data)

	expectComposedOfIsMissing(t, data)
}

func expectValidParsedDetailedTemplatesCommon(t *testing.T, data metricset.FetcherData[IndexTemplates]) {
	events := data.Reporter.GetEvents()

	require.Equal(t, 2, len(events))

	event1 := auto_ops_testing.GetEventByName(t, events, "template.templateName", "simple-response")
	event2 := auto_ops_testing.GetEventByName(t, events, "template.templateName", "detailed-response")

	auto_ops_testing.CheckEventWithRandomTransactionId(t, event2, data.ClusterInfo)

	simpleMapping, err := utils.DeserializeData[map[string]interface{}]([]byte(getMappingObject(t, "simple-response")))
	require.NoError(t, err)
	simple, err := templateSchema.Apply((*simpleMapping)["index_template"].(map[string]interface{}))
	require.NoError(t, err)

	simpleTemplate := mapstr.M{"template": simple}

	detailedMapping, err := utils.DeserializeData[map[string]interface{}]([]byte(getMappingObject(t, "detailed-response")))
	require.NoError(t, err)
	detailed, err := templateSchema.Apply((*detailedMapping)["index_template"].(map[string]interface{}))
	require.NoError(t, err)

	detailedTemplate := mapstr.M{"template": detailed}

	// metrics exist

	// event 1 (simple-response)
	require.Equal(t, "simple-response", auto_ops_testing.GetObjectValue(event1.MetricSetFields, "template.templateName"))
	require.EqualValues(t, 1, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "template.priority"))
	require.Nil(t, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "template.version"))
	require.Nil(t, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "template.allow_auto_create"))
	require.Nil(t, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "template.deprecated"))
	require.ElementsMatch(t, []string{"simple-response"}, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "template.index_patterns"))
	require.Equal(t, auto_ops_testing.GetObjectAsJson(simpleTemplate, "template.template.settings"), auto_ops_testing.GetObjectAsJson(event1.MetricSetFields, "template.template.settings"))
	require.Equal(t, auto_ops_testing.GetObjectAsJson(simpleTemplate, "template.data_stream.hidden"), auto_ops_testing.GetObjectAsJson(event1.MetricSetFields, "template.data_stream.hidden"))
	require.Equal(t, auto_ops_testing.GetObjectAsJson(simpleTemplate, "template.data_stream.allow_custom_routing"), auto_ops_testing.GetObjectAsJson(event1.MetricSetFields, "template.data_stream.allow_custom_routing"))
	require.Nil(t, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "template._meta"))

	// event 2 (detailed-response)
	require.Equal(t, "detailed-response", auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.templateName"))
	require.EqualValues(t, 789, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.priority"))
	require.EqualValues(t, 123456, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.version"))
	require.Equal(t, true, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.allow_auto_create"))
	require.Equal(t, true, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.deprecated"))
	require.ElementsMatch(t, []string{"a", "b", "c", "d*"}, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.index_patterns"))
	require.ElementsMatch(t, []string{"composed-2"}, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.ignore_missing_component_templates"))
	require.EqualValues(t, 789, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.priority"))
	require.Equal(t, auto_ops_testing.GetObjectAsJson(detailedTemplate, "template.template"), auto_ops_testing.GetObjectAsJson(event2.MetricSetFields, "template.template"))
	require.Equal(t, true, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.data_stream.hidden"))
	require.Equal(t, true, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.data_stream.allow_custom_routing"))
	require.EqualValues(t, 456, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template._meta.property1"))

	// schema is expected to drop this field
	require.Nil(t, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "ignored_field"))
	require.Nil(t, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "template.ignored_field"))
	require.Nil(t, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "ignored_field"))
	require.Nil(t, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.ignored_field"))
}

func expectValidParsedDetailedTemplatesOptional(t *testing.T, data metricset.FetcherData[IndexTemplates]) {
	events := data.Reporter.GetEvents()

	event2 := auto_ops_testing.GetEventByName(t, events, "template.templateName", "detailed-response")

	require.ElementsMatch(t, []string{"composed-1", "composed-2"}, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.composed_of"))
}

func expectComposedOfIsMissing(t *testing.T, data metricset.FetcherData[IndexTemplates]) {
	events := data.Reporter.GetEvents()

	event2 := auto_ops_testing.GetEventByName(t, events, "template.templateName", "detailed-response")

	require.Nil(t, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "template.composed_of"))
}

// Tests that Cluster Info is consistently reported and the Templates are properly reported
func expectMixedValidParsedData(t *testing.T, data metricset.FetcherData[IndexTemplates]) {
	require.ErrorContains(t, data.Error, "fetching templates failed for failed-response")
	require.ErrorContains(t, data.Error, "failed applying index template schema for broken-response")

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
	expectedIndexTemplates := []string{"my-index-template-1", "my-index-template-2"}

	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/index_template.*.json", auto_ops_testing.SetupSuccessfulTemplateServer(IndexTemplatePath, templatePathPrefix, getTemplateResponse), useNamedMetricSet, expectValidParsedDataWithTemplates(expectedIndexTemplates))
}

func TestProperlyHandlesResponseV8FilteringByNames(t *testing.T) {
	templates.GivenSomeIndexNamesToExclude(t, "entities_v1_*")

	expectedIndexTemplates := []string{"my-index-template-1", "my-index-template-2"}

	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/index_template.8*.json", auto_ops_testing.SetupSuccessfulTemplateServer(IndexTemplatePath, templatePathPrefix, getTemplateResponse), useNamedMetricSet, expectValidParsedDataWithTemplates(expectedIndexTemplates))
}

// Expect a valid response from Elasticsearch to create N events
func TestProperlyHandlesResponseWithMissingTemplates(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/index_template.*.json", auto_ops_testing.SetupSuccessfulTemplateServerWithIgnoredTemplates(IndexTemplatePath, templatePathPrefix, getTemplateResponse, []string{"my-index-template-1"}), useNamedMetricSet, expectValidParsedData)
}

// Expect a valid response from Elasticsearch to create 2 events
func TestProperlyHandlesCustomResponse(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/custom.index_template.*.json", auto_ops_testing.SetupSuccessfulTemplateServerWithIgnoredTemplates(IndexTemplatePath, templatePathPrefix, getTemplateResponse, []string{"ignored-response"}), useNamedMetricSet, expectValidParsedDetailedTemplates)
}

// Expect a valid response from Elasticsearch to create 2 events
func TestProperlyHandlesCustomResponseWhenComposedOfIsMissing(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/optional.index_template.*.json", auto_ops_testing.SetupSuccessfulTemplateServerWithIgnoredTemplates(IndexTemplatePath, templatePathPrefix, getTemplateResponse, []string{"ignored-response"}), useNamedMetricSet, expectValidParsedDetailedTemplatesWithoutComposedOf)
}

// Expect a valid response from Elasticsearch to create N events
func TestProperlyHandlesInnerErrorsInResponse(t *testing.T) {
	t.Setenv(templates.TEMPLATE_BATCH_SIZE_NAME, "1") // automatically unsets/resets after test

	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/mixed.index_template.*.json", auto_ops_testing.SetupSuccessfulTemplateServerWithFailedRequests(IndexTemplatePath, templatePathPrefix, getTemplateResponse, []string{"failed-response"}), useNamedMetricSet, expectMixedValidParsedData)
}

func TestGetIndexPatterns(t *testing.T) {
	tests := []struct {
		name        string
		template    IndexTemplate
		want        []string
		expectError string
	}{
		{
			name: "should throw error when missing index_patterns",
			template: IndexTemplate{
				Name:          "test-index-template",
				IndexTemplate: map[string]interface{}{},
			},
			want:        nil,
			expectError: `index_patterns not found in template "test-index-template"`,
		},
		{
			name: "should throw error when index_patterns is not a slice",
			template: IndexTemplate{
				Name: "wrongtype",
				IndexTemplate: map[string]interface{}{
					"index_patterns": "not-a-slice",
				},
			},
			want:        nil,
			expectError: `index_patterns in template "wrongtype" is not a slice`,
		},
		{
			name: "should throw error when index_patterns contains non-string element",
			template: IndexTemplate{
				Name: "nonstring",
				IndexTemplate: map[string]interface{}{
					"index_patterns": []interface{}{"valid", 123},
				},
			},
			want:        nil,
			expectError: `index_patterns in template "nonstring" contains a non-string element`,
		},
		{
			name: "should return a valid index_patterns",
			template: IndexTemplate{
				Name: "valid",
				IndexTemplate: map[string]interface{}{
					"index_patterns": []interface{}{"log-*", "metrics-*"},
				},
			},
			want:        []string{"log-*", "metrics-*"},
			expectError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.template.getIndexPatterns()
			if tt.expectError != "" {
				if err == nil || err.Error() != tt.expectError {
					t.Errorf("expected error %q, got %v", tt.expectError, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("expected %v, got %v", tt.want, got)
				}
			}
		})
	}
}
