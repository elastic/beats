// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cat_shards

import (
	"slices"

	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
)

type ResolvedIndices struct {
	Name        string      `json:"name"`
	Attributes  []string    `json:"attributes,omitempty"`
	DataStreams interface{} `json:"data_stream,omitempty"`
	Aliases     interface{} `json:"aliases,omitempty"`
}

type ResolvedApiResponse struct {
	Indices []ResolvedIndices `json:"indices"`
}

func getResolvedIndices(m *elasticsearch.MetricSet) (map[string]IndexMetadata, error) {
	if response, err := utils.FetchAPIData[ResolvedApiResponse](m, ResolveIndexPath); err != nil {
		return nil, err
	} else {
		return parseResolvedIndicesResponse(response), nil
	}
}

func parseResolvedIndicesResponse(response *ResolvedApiResponse) map[string]IndexMetadata {
	indexMetadata := make(map[string]IndexMetadata, len(response.Indices))

	for _, index := range response.Indices {
		typeOfIndex := "index"

		aliases := utils.GetStringArrayFromArrayOrSingleValue(index.Aliases)
		dataStreams := utils.GetStringArrayFromArrayOrSingleValue(index.DataStreams)

		if len(dataStreams) > 0 {
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
			aliases:    append(dataStreams, aliases...),
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
