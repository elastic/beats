// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"fmt"
	"math"

	"github.com/elastic/beats/v7/libbeat/common"
)

// HasNonNumericFloat checks if an event has a non-numeric float in the specific key.
// It returns false and an error if the key cannot be found in the event
func HasNonNumericFloat(event common.MapStr, key string) (bool, error) {
	v, err := event.GetValue(key)
	if err != nil {
		return false, fmt.Errorf("getting value for key %s: %w", key, err)
	}

	if v, ok := v.(float64); ok && (math.IsNaN(v) || math.IsInf(v, 0)) {
		return true, nil
	}

	return false, nil
}
