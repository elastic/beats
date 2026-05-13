// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metadata

import (
	"strings"
	"unicode"
)

// ExtractResourceID extracts the resource identifier from an event identifier.
// Event identifier format: {accountId}-{resourceId}-{index}
// Account ID is always 12 digits, so we detect and strip it.
func ExtractResourceID(eventIdentifier string) string {
	parts := strings.Split(eventIdentifier, "-")
	if len(parts) < 2 {
		return eventIdentifier
	}

	startIdx := 0
	// Check if first part is a 12-digit account ID
	if len(parts[0]) == 12 && isAllDigits(parts[0]) {
		startIdx = 1
	}

	// Remove the last part (index) and join the rest
	if startIdx < len(parts)-1 {
		return strings.Join(parts[startIdx:len(parts)-1], "-")
	}
	return eventIdentifier
}

// isAllDigits returns true if the string contains only digits
func isAllDigits(s string) bool {
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}
