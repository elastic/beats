// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package usage

import "strings"

// dateFormatForStateStore is used to parse and format dates in the YYYY-MM-DD format
const dateFormatForStateStore = "2006-01-02"

func ptr[T any](value T) *T {
	return &value
}

func processHeaders(headers []string) map[string]string {
	headersMap := make(map[string]string, len(headers))
	for _, header := range headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) != 2 {
			continue
		}
		k, v := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if k == "" || v == "" {
			continue
		}
		headersMap[k] = v
	}
	return headersMap
}
