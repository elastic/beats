package management

import (
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

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

	if observed.FeaturesIdx != expectedFeaturesIdx {
		return false
	}

	return true
}
