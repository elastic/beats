package parse

import "github.com/elastic/beats/packetbeat/protos/kafka/internal/kafka"

func RequestHeader(payload []byte) (kafka.RequestHeader, []byte, bool) {
	h, l, ok := readRequestHeader(makeReader(payload))
	if !ok {
		return h, nil, false
	}
	return h, payload[l:], true
}

func ResponseHeader(payload []byte) (kafka.ResponseHeader, []byte, bool) {
	h, l, ok := readResponseHeader(makeReader(payload))
	if !ok {
		return h, nil, false
	}
	return h, payload[l:], true
}

func MetadataRequest(payload []byte) (requ kafka.MetadataRequest, ok bool) {
	buf := makeReader(payload)

	ok = buf.arr(func() {
		if str, ok := buf.string(); ok {
			requ.Topics = append(requ.Topics, str)
		}
	})
	return
}

func MetadataResponse(payload []byte) (resp kafka.MetadataResponse, ok bool) {
	buf := makeReader(payload)

	if ok = buf.arr(func() {
		var bi kafka.BrokerInfo
		bi.ID, _ = buf.id()
		bi.Host, _ = buf.string()
		bi.Port, _ = buf.int32()
		resp.Brokers = append(resp.Brokers, bi)
	}); !ok {
		return
	}

	ok = buf.arr(func() {
		var tm kafka.TopicMetadata
		tm.Error, _ = buf.err()
		tm.Name, _ = buf.string()
		if buf.arr(func() {
			var pm kafka.PartitionMetadata
			pm.Error, _ = buf.err()
			pm.ID, _ = buf.id()
			pm.Leader, _ = buf.id()
			pm.Replicas, _ = buf.idArr()
			pm.ISR, _ = buf.idArr()
			tm.Partitions = append(tm.Partitions, pm)
		}) {
			resp.Topics = append(resp.Topics, tm)
		}
	})
	return
}

func ProduceRequest(payload []byte) (requ kafka.ProduceRequest, ok bool) {
	buf := makeReader(payload)

	requ.RequiredAcks, _ = buf.int16()
	requ.Timeout, _ = buf.int32()
	requ.Topics = map[string]map[kafka.ID]kafka.RawMessageSet{}
	ok = buf.stringMap(func(topic string) {
		partitions := map[kafka.ID]kafka.RawMessageSet{}

		if buf.idMap(func(partition kafka.ID) {
			if msg, ok := buf.messageSet(); ok {
				partitions[partition] = msg
			}
		}) {
			requ.Topics[topic] = partitions
		}
	})
	return
}

func ProduceResponse(v kafka.APIVersion, payload []byte) (resp kafka.ProduceResponse, ok bool) {
	buf := makeReader(payload)

	resp.Topics = map[string]map[kafka.ID]kafka.ProduceResult{}
	if ok = buf.stringMap(func(topic string) {
		partitions := map[kafka.ID]kafka.ProduceResult{}

		if buf.idMap(func(partition kafka.ID) {
			var result kafka.ProduceResult
			result.Error, _ = buf.err()
			result.Offset, _ = buf.int64()
			if v >= kafka.V2 {
				result.Timestamp, _ = buf.int64()
			}

			if !buf.Failed() {
				partitions[partition] = result
			}
		}) {
			resp.Topics[topic] = partitions
		}
	}); !ok {
		return
	}

	if v >= kafka.V1 {
		resp.ThrottleTime, ok = buf.int32()
	}

	return
}

func FetchRequest(payload []byte) (requ kafka.FetchRequest, ok bool) {
	buf := makeReader(payload)

	requ.ReplicaID, _ = buf.id()
	requ.MaxWaitTime, _ = buf.int32()
	requ.MinBytes, ok = buf.int32()
	if !ok {
		return
	}

	requ.Topics = map[string]map[kafka.ID]kafka.FetchParams{}
	ok = buf.stringMap(func(topic string) {
		partitions := map[kafka.ID]kafka.FetchParams{}

		if buf.idMap(func(partition kafka.ID) {
			var params kafka.FetchParams
			params.Offset, _ = buf.int64()
			params.MaxBytes, _ = buf.int32()
			if !buf.Failed() {
				partitions[partition] = params
			}
		}) {
			requ.Topics[topic] = partitions
		}
	})
	return
}

func FetchResponse(v kafka.APIVersion, payload []byte) (resp kafka.FetchResponse, ok bool) {
	buf := makeReader(payload)

	if v >= kafka.V1 {
		resp.ThrottleTime, ok = buf.int32()
		if !ok {
			return
		}
	}

	resp.Topics = map[string]map[kafka.ID]kafka.FetchResult{}
	ok = buf.stringMap(func(topic string) {
		partitions := map[kafka.ID]kafka.FetchResult{}

		if buf.idMap(func(partition kafka.ID) {
			var res kafka.FetchResult
			res.Error, _ = buf.err()
			res.HWMOffset, _ = buf.int64()
			res.MessageSet, _ = buf.messageSet()
			if !buf.Failed() {
				partitions[partition] = res
			}
		}) {
			resp.Topics[topic] = partitions
		}
	})
	return
}

func OffsetRequest(payload []byte) (requ kafka.OffsetRequest, ok bool) {
	buf := makeReader(payload)

	requ.ReplicaID, ok = buf.id()
	if !ok {
		return
	}

	requ.Topics = map[string]map[kafka.ID]kafka.OffsetParams{}
	ok = buf.stringMap(func(topic string) {
		partitions := map[kafka.ID]kafka.OffsetParams{}

		if buf.idMap(func(partition kafka.ID) {
			var params kafka.OffsetParams
			params.Time, _ = buf.int64()
			params.MaxOffsets, _ = buf.int32()
			if !buf.Failed() {
				partitions[partition] = params
			}
		}) {
			requ.Topics[topic] = partitions
		}
	})
	return
}

func OffsetResponse(payload []byte) (resp kafka.OffsetResponse, ok bool) {
	buf := makeReader(payload)

	resp.Topics = map[string]map[kafka.ID]kafka.OffsetResult{}
	ok = buf.stringMap(func(topic string) {
		partitions := map[kafka.ID]kafka.OffsetResult{}

		if buf.idMap(func(partition kafka.ID) {
			var res kafka.OffsetResult
			res.Error, _ = buf.err()
			res.Offsets, _ = buf.int64Arr()
			if !buf.Failed() {
				partitions[partition] = res
			}
		}) {
			resp.Topics[topic] = partitions
		}
	})
	return
}

func GroupCoordinatorRequest(payload []byte) (requ kafka.GroupCoordinatorRequest, ok bool) {
	requ.GroupID, ok = makeReader(payload).string()
	return
}

func GroupCoordinatorResponse(payload []byte) (resp kafka.GroupCoordinatorResponse, ok bool) {
	buf := makeReader(payload)
	resp.Error, _ = buf.err()
	resp.CoordinatorID, _ = buf.id()
	resp.CoordinatorHost, _ = buf.string()
	resp.CoordinatorPort, ok = buf.int32()
	return
}

func OffsetCommitRequest(v kafka.APIVersion, payload []byte) (requ kafka.OffsetCommitRequest, ok bool) {
	buf := makeReader(payload)

	requ.GroupID, ok = buf.string()
	if v >= kafka.V1 {
		requ.GroupGenerationID, _ = buf.id()
		requ.ConsumerID, ok = buf.string()
		if v >= kafka.V2 {
			requ.RetentionTime, ok = buf.int64()
		}
	}
	if !ok {
		return
	}

	requ.Topics = map[string]map[kafka.ID]kafka.CommitParams{}
	ok = buf.stringMap(func(topic string) {
		partitions := map[kafka.ID]kafka.CommitParams{}

		if buf.idMap(func(partition kafka.ID) {
			var params kafka.CommitParams
			params.Offset, _ = buf.int64()
			if v == kafka.V1 {
				params.Timestamp, _ = buf.int64()
			}
			params.Metadata, _ = buf.stringBytes()
			if !buf.Failed() {
				partitions[partition] = params
			}
		}) {
			requ.Topics[topic] = partitions
		}

	})
	return
}

func OffsetCommitResponse(payload []byte) (resp kafka.OffsetCommitResponse, ok bool) {
	buf := makeReader(payload)

	resp.Topics = map[string]map[kafka.ID]kafka.ErrorCode{}
	ok = buf.stringMap(func(topic string) {
		partitions := map[kafka.ID]kafka.ErrorCode{}

		if buf.idMap(func(partition kafka.ID) {
			if err, ok := buf.err(); ok {
				partitions[partition] = err
			}
		}) {
			resp.Topics[topic] = partitions
		}
	})
	return
}

func OffsetFetchRequest(payload []byte) (requ kafka.OffsetFetchRequest, ok bool) {
	buf := makeReader(payload)

	requ.GroupID, ok = buf.string()
	if !ok {
		return
	}

	requ.Topics = map[string][]kafka.ID{}
	ok = buf.stringMap(func(topic string) {
		if partitions, ok := buf.idArr(); ok {
			requ.Topics[topic] = append(requ.Topics[topic], partitions...)
		}
	})
	return
}

func OffsetFetchResponse(payload []byte) (resp kafka.OffsetFetchResponse, ok bool) {
	buf := makeReader(payload)

	resp.Topics = map[string]map[kafka.ID]kafka.OffsetFetchResult{}
	ok = buf.stringMap(func(topic string) {
		partitions := map[kafka.ID]kafka.OffsetFetchResult{}

		if buf.idMap(func(partition kafka.ID) {
			var res kafka.OffsetFetchResult
			res.Offset, _ = buf.int64()
			res.Metadata, _ = buf.stringBytes()
			res.Error, _ = buf.err()
			if !buf.Failed() {
				partitions[partition] = res
			}
		}) {
			resp.Topics[topic] = partitions
		}
	})
	return
}

func JoinGroupRequest(v kafka.APIVersion, payload []byte) (requ kafka.JoinGroupRequest, ok bool) {
	buf := makeReader(payload)

	requ.GroupID, _ = buf.string()
	requ.SessionTimeout, _ = buf.int32()
	if v == kafka.V1 {
		requ.RebalanceTimeout, _ = buf.int32()
	}
	requ.MemberID, _ = buf.string()
	requ.ProtocolType, _ = buf.string()
	requ.Protocols, ok = buf.stringMetaMap()
	return
}

func JoinGroupResponse(payload []byte) (resp kafka.JoinGroupResponse, ok bool) {
	buf := makeReader(payload)
	resp.Error, _ = buf.err()
	resp.GenerationID, _ = buf.int32()
	resp.GroupProtocol, _ = buf.string()
	resp.LeaderID, _ = buf.string()
	resp.MemberID, _ = buf.string()
	resp.Members, ok = buf.stringMetaMap()
	return
}

func SyncGroupRequest(payload []byte) (requ kafka.SyncGroupRequest, ok bool) {
	buf := makeReader(payload)
	requ.GroupID, _ = buf.string()
	requ.GenerationID, _ = buf.int32()
	requ.MemberID, _ = buf.string()
	requ.Assignments, ok = buf.stringMetaMap()
	return
}

func SyncGroupResponse(payload []byte) (resp kafka.SyncGroupResponse, ok bool) {
	buf := makeReader(payload)
	resp.Error, _ = buf.err()
	resp.Assignment, ok = buf.bytes()
	return
}

func HeartbeatRequest(payload []byte) (requ kafka.HeartbeatRequest, ok bool) {
	buf := makeReader(payload)
	requ.GroupID, _ = buf.string()
	requ.GenerationID, _ = buf.int32()
	requ.MemberID, ok = buf.string()
	return
}

func HeartbeatResponse(payload []byte) (resp kafka.HeartbeatResponse, ok bool) {
	resp.Error, ok = makeReader(payload).err()
	return
}

func LeaveGroupRequest(payload []byte) (requ kafka.LeaveGroupRequest, ok bool) {
	buf := makeReader(payload)
	requ.GroupID, _ = buf.string()
	requ.MemberID, ok = buf.string()
	return
}

func LeaveGroupResponse(payload []byte) (resp kafka.LeaveGroupResponse, ok bool) {
	resp.Error, ok = makeReader(payload).err()
	return
}

func ListGroupRequest(payload []byte) (requ kafka.ListGroupRequest, ok bool) {
	return kafka.ListGroupRequest{}, len(payload) == 0
}

func ListGroupResponse(payload []byte) (resp kafka.ListGroupResponse, ok bool) {
	buf := makeReader(payload)

	resp.Error, ok = buf.err()

	groups := map[string]string{}
	if buf.stringMap(func(groupID string) {
		if protocolType, ok := buf.string(); ok {
			groups[groupID] = protocolType
		}
	}) {
		resp.Groups = groups
	}
	return
}

func DescribeGroupsRequest(payload []byte) (requ kafka.DescribeGroupsRequest, ok bool) {
	buf := makeReader(payload)

	ok = buf.arr(func() {
		if id, ok := buf.string(); ok {
			requ.Groups = append(requ.Groups, id)
		}
	})
	return
}

func DescribeGroupsResponse(payload []byte) (resp kafka.DescribeGroupsResponse, ok bool) {
	buf := makeReader(payload)

	resp.Groups = map[string]kafka.GroupMetadata{}
	ok = buf.arr(func() {
		var group kafka.GroupMetadata

		group.Error, _ = buf.err()
		groupID, _ := buf.string()
		group.State, _ = buf.string()
		group.ProtocolType, _ = buf.string()
		group.Protocol, _ = buf.string()

		if buf.arr(func() {
			var info kafka.MemberInfo
			var ok bool

			info.MemberID, _ = buf.string()
			info.ClientID, _ = buf.string()
			info.ClientHost, _ = buf.string()
			info.Metadata, _ = buf.bytes()
			info.Assignment, ok = buf.bytes()

			if ok {
				group.Members = append(group.Members, info)
			}
		}) {
			resp.Groups[groupID] = group
		}
	})
	return
}

func MessageSetNext(msgset kafka.RawMessageSet) (int64, []byte, kafka.RawMessageSet, bool) {
	if len(msgset.Payload) == 0 {
		return -1, nil, msgset, true
	}

	buf := makeReader(msgset.Payload)
	offset, _ := buf.int64()
	msg, ok := buf.bytes()
	if !ok {
		return -1, nil, kafka.RawMessageSet{}, false
	}

	len := buf.BufferConsumed()
	return offset, msg, kafka.RawMessageSet{msgset.Payload[len:]}, true
}

func MessageSetFirst(msgset kafka.RawMessageSet) (int64, []byte, bool) {
	offset, msg, _, ok := MessageSetNext(msgset)
	return offset, msg, ok
}

func IterMessageSet(msgset kafka.RawMessageSet, f func(int64, []byte) bool) bool {
	buf := makeReader(msgset.Payload)

	for buf.Len() > 0 {
		offset, _ := buf.int64()
		msg, ok := buf.bytes()
		if !ok || !f(offset, msg) {
			return false
		}
	}
	return true
}

func CountMessageSetElements(msgset kafka.RawMessageSet) (sz int, ok bool) {
	ok = IterMessageSet(msgset, func(_ int64, _ []byte) bool {
		sz++
		return true
	})
	return
}

func readRequestHeader(buf *protoReader) (header kafka.RequestHeader, len int, ok bool) {
	key, _ := buf.int16()
	header.APIKey = kafka.APIKey(key)
	header.Version, _ = buf.version()
	header.CorrelationId, _ = buf.id()
	header.ClientID, ok = buf.stringBytes()
	len = buf.BufferConsumed()
	return
}

func readResponseHeader(buf *protoReader) (header kafka.ResponseHeader, len int, ok bool) {
	len = 4
	header.CorrelationID, ok = buf.id()
	return
}
