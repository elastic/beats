// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build integration

package integration

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// GetEventsMsgFromES gets the 'message' field from all documents
// in `index`. If Elasticsearch returns an status code other than 200
// nil is returned. `size` sets the number of documents returned
func GetEventsMsgFromES(t *testing.T, index string, size int) []string {
	t.Helper()
	// Step 1: Get the Elasticsearch Admin URL so we can query any index
	esURL := GetESAdminURL(t, "http")

	// Step 2: Format the search URL for the `foo` datastream
	searchURL, err := FormatDataStreamSearchURL(t, esURL, index)
	require.NoError(t, err, "Failed to format datastream search URL")

	// Step 3: Add query parameters
	queryParams := searchURL.Query()

	// Add the `size` (the number of documents returned) parameter
	queryParams.Set("size", strconv.Itoa(size))
	// Order the events in ascending order
	queryParams.Set("sort", "@timestamp:asc")
	// Only request the field we need
	queryParams.Set("_source", "message")
	searchURL.RawQuery = queryParams.Encode()

	// Step 4: Perform the HTTP GET request using HttpDo
	statusCode, body, err := HttpDo(t, "GET", searchURL)
	require.NoError(t, err, "Failed to perform HTTP request")
	if statusCode != 200 {
		return nil
	}

	// Step 5: Parse the response body to extract events
	var searchResult struct {
		Hits struct {
			Hits []struct {
				Source struct {
					Message string `json:"message"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	err = json.Unmarshal(body, &searchResult)
	require.NoError(t, err, "Failed to parse response body")

	// Step 6: Extract the `message` field from each event and return the messages
	messages := []string{}
	for _, hit := range searchResult.Hits.Hits {
		messages = append(messages, hit.Source.Message)
	}

	return messages
}
