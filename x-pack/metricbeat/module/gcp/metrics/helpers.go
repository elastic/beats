// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metrics

import "strings"

// withSuffix ensures a string end with the specified suffix.
func withSuffix(s, suffix string) string {
	if strings.HasSuffix(s, suffix) {
		return s
	}

	return s + suffix
}
