// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package netflow

import (
	"unicode"
)

// CamelCaseToSnakeCase converts a camel-case identifier to snake-case
// format. This function is tailored to some specifics of NetFlow field names.
// Don't reuse it.
func CamelCaseToSnakeCase(in string) string {
	out := make([]rune, 0, len(in)+4)
	runes := []rune(in)
	upperStrike := 1
	for pos, r := range runes {
		lr := unicode.ToLower(r)
		isUpper := lr != r
		if isUpper {
			if upperStrike == 0 {
				out = append(out, '_')
			}
			upperStrike++
		} else {
			if upperStrike > 2 {
				// Some magic here:
				// NetFlow usually lowercases all but the first letter of an
				// acronym (Icmp) Except when it is 2 characters long: (IP).
				// In other cases, it keeps all caps, but if we have a run of
				// more than 2 uppercase chars, then the last char belongs to
				// the next word:
				// postNATSourceIPv4Address     : post_nat_source_ipv4_address
				// selectorIDTotalFlowsObserved : selector_id_total_flows_...
				out = append(out, '_')
				out[pos], out[pos-1] = out[pos-1], out[pos]
			}
			upperStrike = 0
		}
		out = append(out, lr)
	}
	return string(out)
}
