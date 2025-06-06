// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package templates

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"

	"golang.org/x/exp/maps"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var defaultExcludedTemplatePatterns = []string{
	".ml*",
	".monitoring*",
	".transform*",
	".kibana*",
	".security*",
	".apm*",
	".async*",
	".reporting*",
	".enrich*",
	".fleet*",
	".snapshot*",
	".watches*",
	".geoip*",
	".management*",
	".watch*",
	".triggered*",
	".logstash-management*",
	".lists-default*",
	".slm-history*",
	".alerts*",
	".deprecation*",
	".logs*",
	".slm*",
	".slo*",
	"apm*",
	"behavioral_analytics*",
	"elastic*",
	"ilm-history*",
	"logs-apm*",
	"logs-app_search*",
	"logs-crawler*",
	"logs-elastic*",
	"logs-endpoint*",
	"logs-enterprise_search*",
	"logs-fleet_server*",
	"logs-workplace_search*",
	"metrics-apm*",
	"metrics-elasticsearch*",
	"metrics-endpoint*",
	"metrics-fleet_server*",
	"metrics-metadata*",
	"search-*",
	"synthetics*",
	"traces-apm*",
	".entities*",
}

const (
	EXCLUDE_STRING_IN_TEMPLATE_NAMES_NAME       = "EXCLUDE_STRING_IN_TEMPLATE_NAMES"
	IGNORE_TEMPLATES_BY_NAME_NAME               = "IGNORE_TEMPLATES_BY_NAME"
	IGNORE_TEMPLATES_BY_INDEX_PATTERN_NAME_NAME = "IGNORE_TEMPLATES_BY_INDEX_PATTERN_NAME"
	TEMPLATE_BATCH_SIZE_NAME                    = "TEMPLATE_BATCH_SIZE"
	TEMPLATES_SLEEP_INTERVAL_IN_MILLIS_NAME     = "TEMPLATES_SLEEP_INTERVAL_IN_MILLIS"
)

var TemplatesSleepIntervalInMillis = utils.GetIntEnvParam(TEMPLATES_SLEEP_INTERVAL_IN_MILLIS_NAME, 0)

var TemplateIndexPatternsToIgnore = GetTemplateIndexPatternsToFilterOut()

var TemplateIndexNamesToIgnore = GetTemplateNamesToFilterOut()

type FilterTemplate[T any] func(pattern []string) utils.Predicate[T]

type ReportNamedTemplate func(transactionId string, info *utils.ClusterInfo, template mapstr.M)

type GetNamedTemplates[T any] func(transactionId string, info *utils.ClusterInfo, templates *T, reporter ReportNamedTemplate) (errs []error)

// Exposed as a function for testing. This should not change at runtime.
func GetTemplateNamesToFilterOut() []string {
	return SplitOrNil(utils.GetStrenv(IGNORE_TEMPLATES_BY_NAME_NAME, ""))
}

func GetTemplateIndexPatternsToFilterOut() []string {
	envVar, set := os.LookupEnv(IGNORE_TEMPLATES_BY_INDEX_PATTERN_NAME_NAME)
	if set && strings.TrimSpace(envVar) == "" {
		return []string{}
	}
	return SplitOrDefault(envVar, defaultExcludedTemplatePatterns)
}

// Group the `templates` that match the `filter` by name, and create partitions (batches) to fetch.
func GetPartitionedTemplates[T any](templates []T, namer utils.Supplier[*T, string], filter utils.Predicate[*T]) [][]string {
	templateNames := utils.FilterAndMap(templates, namer, func(template *T) bool {
		return filter(template) && !utils.MatchesAnyPattern(namer(template), TemplateIndexNamesToIgnore)
	})

	nameLimit := utils.GetIntEnvParam(TEMPLATE_BATCH_SIZE_NAME, 1500)

	return maps.Values(utils.PartitionByMaxValue(nameLimit, templateNames, func(template string) int {
		return len(template)
	}))
}

func GetPartitionedTemplatesWithErrors[T any](templates []T, namer utils.Supplier[*T, string], filter utils.CheckedPredicate[*T]) ([][]string, []error) {
	var predicateErrors []error
	templateNames := utils.FilterAndMap(templates, namer, func(template *T) bool {
		check, err := filter(template)
		if err != nil {
			predicateErrors = append(predicateErrors, err)
			return false
		}
		return check && !utils.MatchesAnyPattern(namer(template), TemplateIndexNamesToIgnore)
	})

	nameLimit := utils.GetIntEnvParam(TEMPLATE_BATCH_SIZE_NAME, 1500)

	return maps.Values(utils.PartitionByMaxValue(nameLimit, templateNames, func(template string) int {
		return len(template)
	})), predicateErrors
}

// Loop across the `partitionedTemplates` and request them in associated batches, then extract and report them as events sharing a Transaction ID.
func HandlePartitionedTemplates[T any](m *elasticsearch.MetricSet, r mb.ReporterV2, info *utils.ClusterInfo, templatePathPrefix string, partitionedTemplates [][]string, getNamedTemplates GetNamedTemplates[T]) (string, error) {
	var errs []error
	lastIndex := len(partitionedTemplates) - 1

	excludeTemplateSubstring := utils.GetStrenv(EXCLUDE_STRING_IN_TEMPLATE_NAMES_NAME, "%{")

	transactionId := utils.NewUUIDV4()

	for i, templatesBatch := range partitionedTemplates {
		if len(templatesBatch) == 0 {
			continue
		}

		templateNames := utils.UrlEscapeNames(templatesBatch, excludeTemplateSubstring)

		templatesSuffix := strings.Join(templateNames, ",")
		templates, err := utils.FetchAPIData[T](m, templatePathPrefix+templatesSuffix)

		if err != nil {
			errs = append(errs, fmt.Errorf("fetching templates failed for %v: %w", templatesSuffix, err))
			continue
		}

		namedTemplatesErrs := getNamedTemplates(transactionId, info, templates, func(transactionId string, info *utils.ClusterInfo, template mapstr.M) {
			r.Event(events.CreateEvent(info, mapstr.M{"template": template}, transactionId))
		})

		errs = append(errs, namedTemplatesErrs...)

		if i != lastIndex && TemplatesSleepIntervalInMillis > 0 {
			time.Sleep(time.Duration(TemplatesSleepIntervalInMillis) * time.Millisecond)
		}
	}

	return transactionId, errors.Join(errs...)
}

// Loop across the `partitionedTemplates` and request them individually, then extract and report them as events sharing a Transaction ID.
func HandleIndividualTemplateRequests[T any](m *elasticsearch.MetricSet, r mb.ReporterV2, info *utils.ClusterInfo, templatePathPrefix string, partitionedTemplates [][]string, getNamedTemplates GetNamedTemplates[T]) error {
	var errs []error
	lastIndex := len(partitionedTemplates) - 1

	excludeTemplateSubstring := utils.GetStrenv(EXCLUDE_STRING_IN_TEMPLATE_NAMES_NAME, "%{")

	transactionId := utils.NewUUIDV4()

	for i, templatesBatch := range partitionedTemplates {
		if len(templatesBatch) == 0 {
			continue
		}

		templateNames := utils.UrlEscapeNames(templatesBatch, excludeTemplateSubstring)

		for _, templateName := range templateNames {
			apiPath := fmt.Sprintf("%s%s", templatePathPrefix, templateName)

			templateData, err := utils.FetchAPIData[T](m, apiPath)
			if err != nil {
				errs = append(errs, fmt.Errorf("fetching templates failed for %v: %w", templateName, err))
				continue
			}

			namedTemplatesErrs := getNamedTemplates(transactionId, info, templateData, func(transactionId string, info *utils.ClusterInfo, template mapstr.M) {
				r.Event(events.CreateEvent(info, mapstr.M{"template": template}, transactionId))
			})

			errs = append(errs, namedTemplatesErrs...)
		}

		if i != lastIndex && TemplatesSleepIntervalInMillis > 0 {
			time.Sleep(time.Duration(TemplatesSleepIntervalInMillis) * time.Millisecond)
		}
	}

	return errors.Join(errs...)
}

// Get the pattern split by a comma, or return nil if the pattern is empty.
func SplitOrNil(pattern string) []string {
	if len(pattern) > 0 {
		return strings.Split(pattern, ",")
	}

	return nil
}

func SplitOrDefault(pattern string, defaultValue []string) []string {
	pattern = strings.TrimSpace(pattern)

	if len(pattern) > 0 {
		parts := strings.Split(pattern, ",")
		var result []string

		for _, part := range parts {
			trimmedPart := strings.TrimSpace(part)
			if trimmedPart != "" {
				result = append(result, trimmedPart)
			}
		}
		if len(result) > 0 {
			return result
		}
	}

	return defaultValue
}
