// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

// DoesStateMatch returns true if the observed state matches
// the expectedUnits and Features.
func DoesStateMatch(
	observed *proto.CheckinObserved,
	expectedUnits []*proto.UnitExpected,
	expectedFeaturesIdx uint64,
) bool {
	if len(observed.Units) != len(expectedUnits) {
		return false
	}
	expectedMap := make(map[unitKey]*proto.UnitExpected)
	for _, exp := range expectedUnits {
		expectedMap[unitKey{client.UnitType(exp.Type), exp.Id}] = exp
	}
	for _, unit := range observed.Units {
		exp, ok := expectedMap[unitKey{client.UnitType(unit.Type), unit.Id}]
		if !ok {
			return false
		}
		if unit.State != exp.State || unit.ConfigStateIdx != exp.ConfigStateIdx {
			return false
		}
	}

	return observed.FeaturesIdx == expectedFeaturesIdx
}
