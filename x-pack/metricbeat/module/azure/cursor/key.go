// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package cursor

import (
	"fmt"
	"strings"

	"github.com/cespare/xxhash/v2"
)

// GenerateStateKey returns a stable, opaque key for storing cursor state.
// Each argument is length-prefixed before hashing so no part can bleed into
// the next (e.g. "ab"+"c" ≠ "a"+"bc"), and no secrets appear in the stored key.
//
// Key reset triggers — changing any of the following resets the cursor:
//   - metricsetName  — different metricset, different data stream
//   - subscriptionID — different Azure environment
//   - resourcesKey   — fingerprint of metric namespaces and resource-listing
//     filters (resource_id, resource_group, resource_type, resource_query);
//     two configs that collect different namespaces or target different
//     resources are different series
//
// Changing lookback_window, period, or latency alone does NOT reset the cursor.
func GenerateStateKey(metricsetName, subscriptionID, resourcesKey string) string {
	var b strings.Builder
	for _, p := range []string{metricsetName, subscriptionID, resourcesKey} {
		fmt.Fprintf(&b, "%d:%s|", len(p), p)
	}
	hash := xxhash.Sum64String(b.String())
	return fmt.Sprintf("azure-cursor::%x", hash)
}
