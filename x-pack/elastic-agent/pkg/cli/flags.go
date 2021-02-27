// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cli

import "strings"

const splitOn = ","

// StringToSlice takes a string retrieve from a flag and return a slices splitted on comma and every
// element has been trim of space.
func StringToSlice(s string) []string {
	if len(s) == 0 {
		return make([]string, 0)
	}

	elements := strings.Split(s, splitOn)
	for i, v := range elements {
		elements[i] = strings.TrimSpace(v)
	}

	return elements
}
