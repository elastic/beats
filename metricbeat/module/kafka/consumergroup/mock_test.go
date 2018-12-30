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

package consumergroup

import (
	"fmt"
	"math/rand"

	"github.com/Shopify/sarama"

	"github.com/elastic/beats/metricbeat/module/kafka"
)

type mockClient struct {
	listGroups        func() ([]string, error)
	describeGroups    func(group []string) (map[string]kafka.GroupDescription, error)
	fetchGroupOffsets func(group string) (*sarama.OffsetFetchResponse, error)
}

type mockState struct {
	// group -> topics -> partitions -> offset
	partitions map[string]map[string][]int64 // topics with group partition offsets

	// groups->client->topic->partitions ids
	groups map[string][]map[string][]int32 // group/client assignments to topics and partition IDs
}

func defaultMockClient(state mockState) *mockClient {
	return &mockClient{
		listGroups:        makeListGroups(state),
		describeGroups:    makeDescribeGroups(state),
		fetchGroupOffsets: makeFetchGroupOffsets(state),
	}
}

func (c *mockClient) with(fn func(*mockClient)) *mockClient {
	fn(c)
	return c
}

func makeListGroups(state mockState) func() ([]string, error) {
	names := make([]string, 0, len(state.groups))
	for name := range state.groups {
		names = append(names, name)
	}

	return func() ([]string, error) {
		return names, nil
	}
}

func makeDescribeGroups(
	state mockState,
) func([]string) (map[string]kafka.GroupDescription, error) {
	groups := map[string]kafka.GroupDescription{}
	for name, st := range state.groups {
		members := map[string]kafka.MemberDescription{}
		for i, member := range st {
			clientID := fmt.Sprintf("consumer-%v", i)
			memberID := fmt.Sprintf("%v-%v", clientID, rand.Int())
			members[memberID] = kafka.MemberDescription{
				ClientID:   clientID,
				ClientHost: "/" + clientID,
				Topics:     member,
			}
		}
		groups[name] = kafka.GroupDescription{Members: members}
	}

	return func(group []string) (map[string]kafka.GroupDescription, error) {
		ret := map[string]kafka.GroupDescription{}
		for _, name := range group {
			if g, found := groups[name]; found {
				ret[name] = g
			}
		}

		if len(ret) == 0 {
			ret = nil
		}
		return ret, nil
	}
}

func makeDescribeGroupsFail(
	err error,
) func([]string) (map[string]kafka.GroupDescription, error) {
	return func(_ []string) (map[string]kafka.GroupDescription, error) {
		return nil, err
	}
}

func makeFetchGroupOffsets(
	state mockState,
) func(group string) (*sarama.OffsetFetchResponse, error) {
	return func(group string) (*sarama.OffsetFetchResponse, error) {
		topics := state.partitions[group]
		if topics == nil {
			return &sarama.OffsetFetchResponse{}, nil
		}

		blocks := map[string]map[int32]*sarama.OffsetFetchResponseBlock{}
		for topic, partition := range topics {
			T := map[int32]*sarama.OffsetFetchResponseBlock{}
			blocks[topic] = T

			for i, offset := range partition {
				T[int32(i)] = &sarama.OffsetFetchResponseBlock{
					Offset: int64(offset),
				}
			}
		}

		return &sarama.OffsetFetchResponse{Blocks: blocks}, nil
	}
}

func makeFetchGroupOffsetsFail(
	err error,
) func(string) (*sarama.OffsetFetchResponse, error) {
	return func(_ string) (*sarama.OffsetFetchResponse, error) {
		return nil, err
	}
}

func (c *mockClient) ListGroups() ([]string, error) { return c.listGroups() }
func (c *mockClient) DescribeGroups(groups []string) (map[string]kafka.GroupDescription, error) {
	return c.describeGroups(groups)
}
func (c *mockClient) FetchGroupOffsets(group string, partitions map[string][]int32) (*sarama.OffsetFetchResponse, error) {
	return c.fetchGroupOffsets(group)
}
