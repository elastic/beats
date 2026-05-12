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
// The key is derived from the metricset name and subscription ID, length-prefixed
// and hashed so no secrets appear in the stored key.
//
// Key reset: changing subscriptionID resets the cursor (different Azure env).
// Changing lookback_window alone does NOT reset the cursor.
func GenerateStateKey(metricsetName, subscriptionID string) string {
	var b strings.Builder
	for _, p := range []string{metricsetName, subscriptionID} {
		fmt.Fprintf(&b, "%d:%s|", len(p), p)
	}
	hash := xxhash.Sum64String(b.String())
	return fmt.Sprintf("azure-cursor::%x", hash)
}
