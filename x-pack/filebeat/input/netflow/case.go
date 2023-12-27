// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package netflow

import (
	"strings"
	"sync"
	"unicode"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/record"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var fieldNameConverter = caseConverter{
	conversion: map[string]string{
		// Special handled fields
		// VRFname should be VRFName
		"VRFname": "vrf_name",
	},
}

type caseConverter struct {
	rwMutex    sync.RWMutex
	conversion map[string]string
}

func (c *caseConverter) memoize(nfName, converted string) string {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()
	c.conversion[nfName] = converted
	return converted
}

func (c *caseConverter) ToSnakeCase(orig record.Map) mapstr.M {
	result := mapstr.M(make(map[string]interface{}, len(orig)))
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()

	for nfName, value := range orig {
		name, found := c.conversion[nfName]
		if !found {
			c.rwMutex.RUnlock()
			name = c.memoize(nfName, CamelCaseToSnakeCase(nfName))
			c.rwMutex.RLock()
		}
		result[name] = value
	}
	return result
}

// CamelCaseToSnakeCase converts a camel-case identifier to snake-case
// format. This function is tailored to some specifics of NetFlow field names.
// Don't reuse it.
func CamelCaseToSnakeCase(in string) string {
	// skip those few fields that are already snake-cased
	if strings.ContainsRune(in, '_') {
		return strings.ToLower(in)
	}

	out := make([]rune, 0, len(in)+4)
	runes := []rune(in)
	upperCount := 1
	for _, r := range runes {
		lr := unicode.ToLower(r)
		isUpper := lr != r
		if isUpper {
			if upperCount == 0 {
				out = append(out, '_')
			}
			upperCount++
		} else {
			if upperCount > 2 {
				// Some magic here:
				// NetFlow usually lowercases all but the first letter of an
				// acronym (Icmp) Except when it is 2 characters long: (IP).
				// In other cases, it keeps all caps, but if we have a run of
				// more than 2 uppercase chars, then the last char belongs to
				// the next word:
				// postNATSourceIPv4Address     : post_nat_source_ipv4_address
				// selectorIDTotalFlowsObserved : selector_id_total_flows_...
				out = append(out, '_')
				n := len(out) - 1
				out[n], out[n-1] = out[n-1], out[n]
			}
			upperCount = 0
		}
		out = append(out, lr)
	}
	return string(out)
}
