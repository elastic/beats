package integration

import (
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

// unitKey is used to identify a unique unit in a map
// the `ID` of a unit in itself is not unique without its type, only `Type` + `ID` is unique
type unitKey struct {
	Type client.UnitType
	ID   string
}

func NewMockServer(
	units [][]*proto.UnitExpected,
	featuresIdxs []uint64,
	features []*proto.Features,
	observedCallback func(*proto.CheckinObserved, int),
	delay time.Duration,
) *mock.StubServerV2 {
	i := 0
	agentInfo := &proto.CheckinAgentInfo{
		Id:       "elastic-agent-id",
		Version:  version.GetDefaultVersion(),
		Snapshot: true,
	}
	return &mock.StubServerV2{
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
			if observedCallback != nil {
				observedCallback(observed, i)
			}
			matches := DoesStateMatch(observed, units[i], featuresIdxs[i])
			if !matches {
				// send same set of units and features
				return &proto.CheckinExpected{
					AgentInfo:   agentInfo,
					Units:       units[i],
					Features:    features[i],
					FeaturesIdx: featuresIdxs[i],
				}
			}
			// delay sending next expected based on delay
			if delay > 0 {
				<-time.After(delay)
			}
			// send next set of units and features
			i += 1
			if i >= len(units) {
				// stay on last index
				i = len(units) - 1
			}
			return &proto.CheckinExpected{
				AgentInfo:   agentInfo,
				Units:       units[i],
				Features:    features[i],
				FeaturesIdx: featuresIdxs[i],
			}
		},
		ActionImpl: func(response *proto.ActionResponse) error {
			// actions not tested here
			return nil
		},
		ActionsChan: make(chan *mock.PerformAction, 100),
	}
}

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

func RequireNewStruct(t *testing.T, v map[string]interface{}) *structpb.Struct {
	str, err := structpb.NewStruct(v)
	if err != nil {
		require.NoError(t, err, "could not convert map[string]interface{} into structpb")
	}
	return str
}
