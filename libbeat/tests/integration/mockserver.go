// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

// unitKey is used to identify a unique unit in a map
// the `ID` of a unit in itself is not unique without its type, only `Type` + `ID` is unique
type unitKey struct {
	Type client.UnitType
	ID   string
}

// NewMockServer creates a GRPC server to mock the Elastic-Agent.
// On the first check-in call it will send the first element of `unit`
// as the expected unit, on successive calls, if the Beat has reached
// that state, it will move on to sending the next state.
// It will also validate the features.
//
// if `observedCallback` is not nil, it will be called on every
// check-in receiving the `proto.CheckinObserved` sent by the
// Beat and index from `units` that was last sent to the Beat.
//
// If `delay` is not zero, when the Beat state matches the last
// sent units, the server will wait for `delay` before sending the
// next state. This will block the check-in call from the Beat.
func NewMockServer(
	units [][]*proto.UnitExpected,
	featuresIdxs []uint64,
	features []*proto.Features,
	observedCallback func(*proto.CheckinObserved, int),
	delay time.Duration,
) *mock.StubServerV2 {
	i := 0
	agentInfo := &proto.AgentInfo{
		Id:       "elastic-agent-id",
		Version:  version.GetDefaultVersion(),
		Snapshot: true,
	}
	return &mock.StubServerV2{
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
			if observedCallback != nil {
				observedCallback(observed, i)
			}
			matches := doesStateMatch(observed, units[i], featuresIdxs[i])
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

func doesStateMatch(
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
