// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureeventhub

import "strings"

type sanitizationFunc func(jsonStr string) []byte

func getSanitizationFuncs() map[string]sanitizationFunc {
	return map[string]sanitizationFunc{
		"NEW_LINES":     sanitizeNewLines,
		"SINGLE_QUOTES": sanitizeSingleQuotes,
	}
}

func sanitize(jsonStr string, opts ...string) []byte {
	var res []byte

	for _, opt := range opts {
		f := getSanitizationFuncs()[opt]
		res = f(jsonStr)
	}

	return res
}

func sanitizeNewLines(jsonStr string) []byte {
	var result strings.Builder

	for _, r := range jsonStr {
		if r == '\n' {
			continue
		}
		result.WriteRune(r)
	}

	return []byte(result.String())
}

func sanitizeSingleQuotes(jsonStr string) []byte {
	var result strings.Builder
	inDoubleQuotes := false

	for _, r := range jsonStr {
		if r == '"' {
			inDoubleQuotes = !inDoubleQuotes
		}

		if r == '\'' && !inDoubleQuotes {
			result.WriteRune('"')
		} else {
			result.WriteRune(r)
		}
	}

	return []byte(result.String())
}
