// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cat_shards

import (
	"slices"

	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
)

// Limit aliases reported for an index to reduce event size
// Note: Setting this to 0 will drop all aliases from the event
const MAX_ALIASES_PER_INDEX_NAME string = "MAX_ALIASES_PER_INDEX"

type resolvedIndices struct {
	Name       string   `json:"name"`
	Attributes []string `json:"attributes,omitempty"`
	DataStream string   `json:"data_stream,omitempty"`
	Aliases    []string `json:"aliases,omitempty"`
}

type resolvedApiResponse struct {
	Indices []resolvedIndices `json:"indices"`
}

func getResolvedIndices(m *elasticsearch.MetricSet) (map[string]IndexMetadata, error) {
	if response, err := utils.FetchAPIData[resolvedApiResponse](m, resolveIndexPath); err != nil {
		return nil, err
	} else {
		return parseResolvedIndicesResponse(response), nil
	}
}

func truncateAliases(aliases []string, maxAliases int) []string {
	if len(aliases) > maxAliases {
		return aliases[:maxAliases]
	}
	return aliases
}

func dataStreamAndAliasesCombined(dataStream string, aliases []string, maxAliasesPerIndex int) []string {
	var result []string
	if dataStream != "" {
		result = []string{dataStream}
	}
	return append(result, truncateAliases(aliases, maxAliasesPerIndex)...)
}

func parseResolvedIndicesResponse(response *resolvedApiResponse) map[string]IndexMetadata {
	maxAliasesPerIndex := utils.GetIntEnvParam(MAX_ALIASES_PER_INDEX_NAME, 5)
	indexMetadata := make(map[string]IndexMetadata, len(response.Indices))

	for _, index := range response.Indices {
		typeOfIndex := "index"

		if index.DataStream != "" {
			typeOfIndex = "data_stream"
		}

		attributes := index.Attributes
		attributes, isOpen := deleteFromSlice(attributes, matchAttribute("open"))
		attributes, isSystem := deleteFromSlice(attributes, matchAttribute("system"))
		attributes, isHidden := deleteFromSlice(attributes, matchAttribute("hidden"))

		// don't serialize an empty array
		if len(attributes) == 0 && attributes != nil {
			attributes = nil
		}

		indexMetadata[index.Name] = IndexMetadata{
			indexType:  typeOfIndex,
			open:       isOpen,
			system:     isSystem,
			hidden:     isHidden,
			aliases:    dataStreamAndAliasesCombined(index.DataStream, index.Aliases, maxAliasesPerIndex),
			attributes: attributes,
		}
	}

	return indexMetadata
}

func matchAttribute(attribute string) func(string) bool {
	return func(value string) bool {
		return attribute == value
	}
}

func deleteFromSlice[S ~[]E, E any](slice S, del func(E) bool) (S, bool) {
	size := len(slice)
	newSlice := slices.DeleteFunc(slice, del)

	return newSlice, size != len(newSlice)
}
