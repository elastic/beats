// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package usage

import "strings"

func ptr[T any](value T) *T {
	return &value
}

func processHeaders(headers []string) map[string]string {
	headersMap := make(map[string]string, len(headers))
	for _, header := range headers {
		parts := strings.Split(header, ":")
		if len(parts) != 2 {
			continue
		}
		headersMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return headersMap
}
