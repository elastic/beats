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
	"github.com/Shopify/sarama"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/metricbeat/module/kafka"
)

type client interface {
	ListGroups() ([]string, error)
	DescribeGroups(group []string) (map[string]kafka.GroupDescription, error)
	FetchGroupOffsets(group string, partitions map[string][]int32) (*sarama.OffsetFetchResponse, error)
	FetchPartitionOffsetFromTheLeader(topic string, partitionID int32) (int64, error)
}

func fetchGroupInfo(
	emit func(common.MapStr),
	b client,
	groupsFilter, topicsFilter func(string) bool,
) error {
	type result struct {
		err    error
		group  string
		assign map[string]map[int32]groupAssignment
		off    *sarama.OffsetFetchResponse
	}

	groups, err := listGroups(b, groupsFilter)
	if err != nil {
		logp.Err("failed to list known kafka groups: %v", err)
		return err
	}
	if len(groups) == 0 {
		return nil
	}

	debugf("known consumer groups: ", groups)

	assignments, err := fetchGroupAssignments(b, groups)
	if err != nil {
		logp.Err("failed to fetch kafka group assignments: %v", err)
		return err
	}
	if len(assignments) == 0 {
		return nil
	}

	results := make(chan result)
	waiting := 0
	for group, topics := range assignments {
		// generate the map topic to partitions
		queryTopics := make(map[string][]int32)
		for topic, partitions := range topics {
			if topicsFilter != nil && !topicsFilter(topic) {
				continue
			}

			// copy partition ids
			count := len(partitions)
			if count == 0 {
				continue
			}

			ids, i := make([]int32, count), 0
			for partition := range partitions {
				ids[i], i = partition, i+1
			}
			queryTopics[topic] = ids
		}

		if len(queryTopics) == 0 {
			continue
		}

		// fetch group offset
		waiting++
		go func(group string, partitions map[string][]int32, assign map[string]map[int32]groupAssignment) {
			resp, err := fetchGroupOffset(b, group, partitions)
			if err != nil {
				logp.Err("failed to fetch '%v' group offset: %v", group, err)
			}
			results <- result{err, group, assign, resp}
		}(group, queryTopics, topics)
	}

	for waiting > 0 {
		ret := <-results
		waiting--
		if ret.err != nil && err == nil {
			err = ret.err
		}
		if err != nil {
			continue
		}

		for topic, partitions := range ret.off.Blocks {
			for partition, info := range partitions {
				partitionOffset, err := getPartitionOffsetFromTheLeader(b, topic, partition)
				if err != nil {
					logp.Err("failed to fetch offset for (topic, partition): ('%v', %v)", topic, partition)
					continue
				}
				consumerLag := partitionOffset - info.Offset
				event := common.MapStr{
					"id":           ret.group,
					"topic":        topic,
					"partition":    partition,
					"offset":       info.Offset,
					"meta":         info.Metadata,
					"consumer_lag": consumerLag,
					"error": common.MapStr{
						"code": info.Err,
					},
				}

				if asgnTopic, ok := ret.assign[topic]; ok {
					if assignment, found := asgnTopic[partition]; found {
						event["client"] = common.MapStr{
							"id":        assignment.clientID,
							"host":      assignment.clientHost,
							"member_id": assignment.memberID,
						}
					}
				}
				emit(event)
			}
		}

	}

	close(results)

	return err
}

func getPartitionOffsetFromTheLeader(b client, topic string, partitionID int32) (int64, error) {
	offset, err := b.FetchPartitionOffsetFromTheLeader(topic, partitionID)
	if err != nil {
		return -1, err
	}
	return offset, nil
}

func listGroups(b client, filter func(string) bool) ([]string, error) {
	groups, err := b.ListGroups()
	if err != nil {
		return nil, err
	}

	if filter == nil {
		return groups, nil
	}

	filtered := groups[:0]
	for _, name := range groups {
		if filter(name) {
			filtered = append(filtered, name)
		}
	}
	return filtered, nil
}

func fetchGroupAssignments(
	b client,
	groupIDs []string,
) (map[string]map[string]map[int32]groupAssignment, error) {
	resp, err := b.DescribeGroups(groupIDs)
	if err != nil {
		return nil, err
	}

	groups := map[string]map[string]map[int32]groupAssignment{}

groupLoop:
	for groupID, info := range resp {
		G := groups[groupID]
		if G == nil {
			G = map[string]map[int32]groupAssignment{}
			groups[groupID] = G
		}

		for memberID, memberDescr := range info.Members {
			if memberDescr.Err != nil {
				// group doesn't seem to use standardized member assignment encoding
				// => try next group
				continue groupLoop
			}

			clientID := memberDescr.ClientID
			clientHost := memberDescr.ClientHost
			if len(clientHost) > 1 && clientHost[0] == '/' {
				clientHost = clientHost[1:]
			}

			meta := groupAssignment{
				memberID:   memberID,
				clientID:   clientID,
				clientHost: clientHost,
			}

			for topic, partitions := range memberDescr.Topics {
				T := G[topic]
				if T == nil {
					T = map[int32]groupAssignment{}
					G[topic] = T
				}

				for _, partition := range partitions {
					T[partition] = meta
				}
			}
		}
	}

	return groups, nil
}

func fetchGroupOffset(
	b client,
	group string,
	partitions map[string][]int32,
) (*sarama.OffsetFetchResponse, error) {
	resp, err := b.FetchGroupOffsets(group, partitions)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
