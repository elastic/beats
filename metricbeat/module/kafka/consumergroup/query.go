package consumergroup

import (
	"sync"

	"github.com/Shopify/sarama"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/module/kafka"
)

type client interface {
	ListGroups() ([]string, error)
	DescribeGroups(group []string) (map[string]kafka.GroupDescription, error)
	FetchGroupOffsets(group string) (*sarama.OffsetFetchResponse, error)
}

func fetchGroupInfo(
	emit func(common.MapStr),
	b client,
	groupsFilter, topicsFilter func(string) bool,
) error {
	type result struct {
		err   error
		group string
		off   *sarama.OffsetFetchResponse
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

	wg := sync.WaitGroup{}
	results := make(chan result, len(groups))
	for _, group := range groups {
		group := group

		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := fetchGroupOffset(b, group, topicsFilter)
			if err != nil {
				logp.Err("failed to fetch '%v' group offset: %v", group, err)
			}
			results <- result{err, group, resp}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	assignments, err := fetchGroupAssignments(b, groups)
	if err != nil {
		// wait for workers to stop and drop results
		for range results {
		}

		return err
	}

	for ret := range results {
		if err := ret.err; err != nil {
			// wait for workers to stop and drop results
			for range results {
			}
			return err
		}

		asgnGroup := assignments[ret.group]
		for topic, partitions := range ret.off.Blocks {
			var asgnTopic map[int32]groupAssignment
			if asgnGroup != nil {
				asgnTopic = asgnGroup[topic]
			}

			for partition, info := range partitions {
				event := common.MapStr{
					"id":        ret.group,
					"topic":     topic,
					"partition": partition,
					"offset":    info.Offset,
					"meta":      info.Metadata,
					"error": common.MapStr{
						"code": info.Err,
					},
				}

				if asgnTopic != nil {
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

	return nil
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
	topics func(string) bool,
) (*sarama.OffsetFetchResponse, error) {
	resp, err := b.FetchGroupOffsets(group)
	if err != nil {
		return nil, err
	}

	if topics == nil {
		return resp, err
	}

	for topic := range resp.Blocks {
		if !topics(topic) {
			delete(resp.Blocks, topic)
		}
	}

	return resp, nil
}
