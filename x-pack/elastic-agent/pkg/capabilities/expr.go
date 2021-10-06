// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

import "strings"

const (
	wild      = "*"
	separator = "/"
)

func matchesExpr(pattern, target string) bool {
	if pattern == wild {
		return true
	}

	patternParts := strings.Split(pattern, separator)
	targetParts := strings.Split(target, separator)

	if len(patternParts) != len(targetParts) {
		return false
	}

	for i, pp := range patternParts {
		if pp == wild {
			continue
		}

		if pp != targetParts[i] {
			return false
		}
	}

	return true
}
