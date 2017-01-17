package kafka

import (
	"errors"

	"github.com/pborman/uuid"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/protos/kafka/internal/kafka"
	"github.com/elastic/beats/packetbeat/protos/kafka/internal/kafka/parse"
)

type parser struct {
	handlers [kafka.APITypes]parseHandler

	cb parserEventHandler
}

type parserConfig struct {
	ignoreAPI []string // list of transaction type to ignore
	detailed  bool
}

type parserEventHandler func(
	requMsg *requestMessage,
	respMsg *responseMessage,
	event common.MapStr,
) error

type parseHandler func(
	p *parser,
	txid []byte,
	requMsg *requestMessage,
	respMsg *responseMessage,
) error

var errParseFail = errors.New("parsing failed")

var defaultTransactionHandlers = [kafka.APITypes]parseHandler{
	(*parser).onProduceTransaction,
	(*parser).onFetchTransaction,
	(*parser).onOffsetTransaction,
	(*parser).onMetadataTransaction,
	(*parser).onInternalTransaction,
	(*parser).onInternalTransaction,
	(*parser).onInternalTransaction,
	(*parser).onInternalTransaction,
	(*parser).onOffsetCommitTransaction,
	(*parser).onOffsetFetchTransaction,
	(*parser).onGroupCoordinatorTransaction,
	(*parser).onJoinGroupTransaction,
	(*parser).onHeartbeatTransaction,
	(*parser).onLeaveGroupTransaction,
	(*parser).onSyncGroupTransaction,
	(*parser).onDescribeGroupsTransaction,
	(*parser).onListGroupsTransaction,
}

func newParser(cb parserEventHandler, cfg *parserConfig) *parser {
	p := &parser{
		cb:       cb,
		handlers: defaultTransactionHandlers,
	}

	if !cfg.detailed {
		for i := range p.handlers {
			p.handlers[i] = (*parser).onTransactionNoDetails
		}
	}

	for _, name := range cfg.ignoreAPI {
		key := kafka.APIKeyByName(name)
		if key == kafka.APIKeyInvalid {
			logp.Err("Invalid API method: %v", name)
			continue
		}

		// delete handler in order to drop transactions
		// before actual parsing occurs
		p.handlers[key] = nil
	}

	return p
}

func (p *parser) onTransaction(
	requMsg *requestMessage,
	respMsg *responseMessage,
) error {
	var handler parseHandler
	key := int(requMsg.header.APIKey)
	if 0 <= key && key <= len(p.handlers) {
		handler = p.handlers[key]
	}

	if handler == nil {
		// silently drop kafka transaction
		debugf("Dropping unhandled kafka transaction")
		return nil
	}
	return handler(p, makeTXID(), requMsg, respMsg)
}

func (p *parser) onProduceTransaction(
	txid []byte,
	requMsg *requestMessage,
	respMsg *responseMessage,
) error {
	requ, ok := parse.ProduceRequest(requMsg.payload)
	if !ok {
		return errParseFail
	}
	resp, ok := parse.ProduceResponse(requMsg.header.Version, respMsg.payload)
	if !ok {
		return errParseFail
	}

	for topic, topicResp := range resp.Topics {
		requPartition := requ.Topics[topic]
		var notes []string

		if requPartition == nil {
			notes = append(notes, "Missing Topic in Request")
		}

		for partition, result := range topicResp {

			// create event request details
			request := common.MapStr{
				"required_acks": requ.RequiredAcks,
				"timeout":       requ.Timeout,
			}

			if requPartition != nil {
				msgset, exists := requ.Topics[topic][partition]
				if !exists {
					notes = append(notes, "Missing Partition in Request")
				} else {
					if count, ok := parse.CountMessageSetElements(msgset); ok {
						request["messages"] = count
					} else {
						notes = append(notes, "Failed to decode Request MessageSet")
					}
				}
			}

			// create event response details
			response := common.MapStr{
				"error":  getError(result.Error),
				"offset": result.Offset,
			}
			if requMsg.header.Version == kafka.V2 {
				response["timestamp"] = result.Timestamp
			}

			// create event
			event := common.MapStr{
				"status":         getStatus(result.Error),
				"transaction_id": txid,
				"produce": common.MapStr{
					"topic":     topic,
					"partition": partition,
					"request":   request,
					"response":  response,
				},
			}
			if len(notes) > 0 {
				event["notes"] = notes
			}

			p.cb(requMsg, respMsg, event)
		}
	}

	return nil
}

func (p *parser) onFetchTransaction(
	txid []byte,
	requMsg *requestMessage,
	respMsg *responseMessage,
) error {
	requ, ok := parse.FetchRequest(requMsg.payload)
	if !ok {
		return errParseFail
	}
	resp, ok := parse.FetchResponse(requMsg.header.Version, respMsg.payload)
	if !ok {
		return errParseFail
	}

	for topic, topicResp := range resp.Topics {
		requPartition := requ.Topics[topic]
		var notes []string

		if requPartition == nil {
			notes = append(notes, "Missing Topic in Request")
		}

		for partition, result := range topicResp {

			// create event request details
			request := common.MapStr{
				"replica_id":    requ.ReplicaID,
				"max_wait_time": requ.MaxWaitTime,
				"min_bytes":     requ.MinBytes,
			}

			if requPartition != nil {
				params, exists := requ.Topics[topic][partition]
				if !exists {
					notes = append(notes, "Missing Partition in Request")
				} else {
					request["offset"] = params.Offset
					request["max_bytes"] = params.MaxBytes
				}
			}

			// create event response details

			response := common.MapStr{
				"error":      getError(result.Error),
				"hwm_offset": result.HWMOffset,
			}
			if count, ok := parse.CountMessageSetElements(result.MessageSet); ok {
				response["messages"] = count
			} else {
				notes = append(notes, "Failed to decode Response MessageSet")
			}

			// create event
			event := common.MapStr{
				"status":         getStatus(result.Error),
				"transaction_id": txid,
				"fetch": common.MapStr{
					"topic":     topic,
					"partition": partition,
					"request":   request,
					"response":  response,
				},
			}
			if len(notes) > 0 {
				event["notes"] = notes
			}

			p.cb(requMsg, respMsg, event)
		}
	}
	return nil
}

func (p *parser) onOffsetTransaction(
	txid []byte,
	requMsg *requestMessage,
	respMsg *responseMessage,
) error {
	requ, ok := parse.OffsetRequest(requMsg.payload)
	if !ok {
		return errParseFail
	}
	resp, ok := parse.OffsetResponse(respMsg.payload)
	if !ok {
		return errParseFail
	}

	for topic, topicResp := range resp.Topics {
		requPartition := requ.Topics[topic]
		var notes []string

		if requPartition == nil {
			notes = append(notes, "Missing Topic in Request")
		}

		for partition, result := range topicResp {

			// create event request details
			request := common.MapStr{
				"replica_id": requ.ReplicaID,
			}

			if requPartition != nil {
				params, exists := requ.Topics[topic][partition]
				if !exists {
					notes = append(notes, "Missing Partition in Request")
				} else {
					request["time"] = params.Time
					request["max_offsets"] = params.MaxOffsets
				}
			}

			// create event response details

			response := common.MapStr{
				"error":   getError(result.Error),
				"offsets": result.Offsets,
			}

			// create event
			event := common.MapStr{
				"status":         getStatus(result.Error),
				"transaction_id": txid,
				"offsets": common.MapStr{
					"topic":     topic,
					"partition": partition,
					"request":   request,
					"response":  response,
				},
			}
			if len(notes) > 0 {
				event["notes"] = notes
			}

			p.cb(requMsg, respMsg, event)
		}
	}
	return nil
}

func (p *parser) onMetadataTransaction(
	txid []byte,
	requMsg *requestMessage,
	respMsg *responseMessage,
) error {
	_, ok := parse.MetadataRequest(requMsg.payload)
	if !ok {
		return errParseFail
	}
	resp, ok := parse.MetadataResponse(respMsg.payload)
	if !ok {
		return errParseFail
	}

	for _, broker := range resp.Brokers {
		event := common.MapStr{
			"status":         common.OK_STATUS,
			"transaction_id": txid,
			"metadata": common.MapStr{
				"broker": common.MapStr{
					"host": broker.Host,
					"port": broker.Port,
				},
			},
		}
		p.cb(requMsg, respMsg, event)
	}

	for _, topic := range resp.Topics {
		event := common.MapStr{
			"status":         getStatus(topic.Error),
			"transaction_id": txid,
			"metadata": common.MapStr{
				"topic": common.MapStr{
					"name":       topic.Name,
					"error":      getError(topic.Error),
					"partitions": topic.Partitions,
				},
			},
		}
		p.cb(requMsg, respMsg, event)
	}

	return nil
}

func (p *parser) onInternalTransaction(
	txid []byte,
	requMsg *requestMessage,
	respMsg *responseMessage,
) error {
	event := common.MapStr{
		"status":         "Unknown",
		"transaction_id": txid,
	}
	p.cb(requMsg, respMsg, event)
	return nil
}

func (p *parser) onOffsetCommitTransaction(
	txid []byte,
	requMsg *requestMessage,
	respMsg *responseMessage,
) error {
	debugf("onOffsetCommitTransaction")

	requ, ok := parse.OffsetCommitRequest(requMsg.header.Version, requMsg.payload)
	if !ok {
		debugf("  failed to parse request")
		return errParseFail
	}
	resp, ok := parse.OffsetCommitResponse(respMsg.payload)
	if !ok {
		debugf("  failed to parse response")
		return errParseFail
	}

	for topic, topicResp := range resp.Topics {
		requPartition := requ.Topics[topic]
		var notes []string

		if requPartition == nil {
			notes = append(notes, "Missing Topic in Request")
		}

		for partition, result := range topicResp {

			// create event request details
			request := common.MapStr{
				"group_id":       requ.GroupID,
				"generation_id":  requ.GroupGenerationID,
				"consumer_id":    requ.ConsumerID,
				"retention_time": requ.RetentionTime,
			}

			if requPartition != nil {
				params, exists := requ.Topics[topic][partition]
				if !exists {
					notes = append(notes, "Missing Partition in Request")
				} else {
					request["offset"] = params.Offset
					request["timestamp"] = params.Timestamp
				}
			}

			// create event response details

			response := common.MapStr{
				"error": common.MapStr{
					"code": result,
				},
			}

			// create event
			event := common.MapStr{
				"status":         getStatus(result),
				"transaction_id": txid,
				"offset_commit": common.MapStr{
					"topic":     topic,
					"partition": partition,
					"request":   request,
					"response":  response,
				},
			}
			if len(notes) > 0 {
				event["notes"] = notes
			}

			p.cb(requMsg, respMsg, event)
		}
	}
	return nil
}

func (p *parser) onOffsetFetchTransaction(
	txid []byte,
	requMsg *requestMessage,
	respMsg *responseMessage,
) error {
	requ, ok := parse.OffsetFetchRequest(requMsg.payload)
	if !ok {
		return errParseFail
	}
	resp, ok := parse.OffsetFetchResponse(respMsg.payload)
	if !ok {
		return errParseFail
	}

	for topic, topicResp := range resp.Topics {
		requPartition := requ.Topics[topic]
		var notes []string

		if requPartition == nil {
			notes = append(notes, "Missing Topic in Request")
		}

		for partition, result := range topicResp {

			// create event request details
			request := common.MapStr{
				"group_id": requ.GroupID,
			}

			// create event response details
			response := common.MapStr{
				"offset": result.Offset,
				"error": common.MapStr{
					"code": result.Error,
				},
			}

			// create event
			event := common.MapStr{
				"status":         getStatus(result.Error),
				"transaction_id": txid,
				"offset_fetch": common.MapStr{
					"topic":     topic,
					"partition": partition,
					"request":   request,
					"response":  response,
				},
			}
			if len(notes) > 0 {
				event["notes"] = notes
			}

			p.cb(requMsg, respMsg, event)
		}
	}
	return nil
}

func (p *parser) onGroupCoordinatorTransaction(
	txid []byte,
	requMsg *requestMessage,
	respMsg *responseMessage,
) error {
	requ, ok := parse.GroupCoordinatorRequest(requMsg.payload)
	if !ok {
		return errParseFail
	}
	resp, ok := parse.GroupCoordinatorResponse(respMsg.payload)
	if !ok {
		return errParseFail
	}

	// create event
	event := common.MapStr{
		"status":         getStatus(resp.Error),
		"transaction_id": txid,
		"group_coordinator": common.MapStr{
			"request": common.MapStr{
				"group_id": requ.GroupID,
			},
			"response": common.MapStr{
				"error": getError(resp.Error),
				"coordinator": common.MapStr{
					"id":   resp.CoordinatorID,
					"host": resp.CoordinatorHost,
					"port": resp.CoordinatorPort,
				},
			},
		},
	}

	p.cb(requMsg, respMsg, event)
	return nil
}

func (p *parser) onJoinGroupTransaction(
	txid []byte,
	requMsg *requestMessage,
	respMsg *responseMessage,
) error {
	requ, ok := parse.JoinGroupRequest(requMsg.header.Version, requMsg.payload)
	if !ok {
		return errParseFail
	}
	resp, ok := parse.JoinGroupResponse(respMsg.payload)
	if !ok {
		return errParseFail
	}

	event := common.MapStr{
		"status":         getStatus(resp.Error),
		"transaction_id": txid,
		"join_group": common.MapStr{
			"request": common.MapStr{
				"group_id":          requ.GroupID,
				"session_timeout":   requ.SessionTimeout,
				"rebalance_timeout": requ.RebalanceTimeout,
				"member_id":         requ.MemberID,
				"protocol_type":     requ.ProtocolType,
			},
			"response": common.MapStr{
				"error":          getError(resp.Error),
				"generation_id":  resp.GenerationID,
				"group_protocol": resp.GroupProtocol,
				"leader_id":      resp.LeaderID,
				"member_id":      resp.MemberID,
			},
		},
	}
	p.cb(requMsg, respMsg, event)
	return nil
}

func (p *parser) onHeartbeatTransaction(
	txid []byte,
	requMsg *requestMessage,
	respMsg *responseMessage,
) error {
	requ, ok := parse.HeartbeatRequest(requMsg.payload)
	if !ok {
		return errParseFail
	}
	resp, ok := parse.HeartbeatResponse(respMsg.payload)
	if !ok {
		return errParseFail
	}

	event := common.MapStr{
		"status":         getStatus(resp.Error),
		"transaction_id": txid,
		"heartbeat": common.MapStr{
			"request": common.MapStr{
				"group_id":      requ.GroupID,
				"generation_id": requ.GenerationID,
				"member_id":     requ.MemberID,
			},
			"response": common.MapStr{
				"error": getError(resp.Error),
			},
		},
	}
	p.cb(requMsg, respMsg, event)
	return nil
}

func (p *parser) onLeaveGroupTransaction(
	txid []byte,
	requMsg *requestMessage,
	respMsg *responseMessage,
) error {
	requ, ok := parse.LeaveGroupRequest(requMsg.payload)
	if !ok {
		return errParseFail
	}
	resp, ok := parse.LeaveGroupResponse(respMsg.payload)
	if !ok {
		return errParseFail
	}

	event := common.MapStr{
		"status":         getStatus(resp.Error),
		"transaction_id": txid,
		"leave_group": common.MapStr{
			"request": common.MapStr{
				"group_id":  requ.GroupID,
				"member_id": requ.MemberID,
			},
			"response": common.MapStr{
				"error": getError(resp.Error),
			},
		},
	}
	p.cb(requMsg, respMsg, event)
	return nil
}

func (p *parser) onSyncGroupTransaction(
	txid []byte,
	requMsg *requestMessage,
	respMsg *responseMessage,
) error {
	debugf("onSyncGroupTransaction")

	requ, ok := parse.SyncGroupRequest(requMsg.payload)
	if !ok {
		debugf("  failed parse request")
		return errParseFail
	}
	resp, ok := parse.SyncGroupResponse(respMsg.payload)
	if !ok {
		debugf("  failed parse response")
		return errParseFail
	}

	event := common.MapStr{
		"status":         getStatus(resp.Error),
		"transaction_id": txid,
		"sync_group": common.MapStr{
			"request": common.MapStr{
				"group_id":      requ.GroupID,
				"generation_id": requ.GenerationID,
				"member_id":     requ.MemberID,
			},
			"response": common.MapStr{
				"error": getError(resp.Error),
			},
		},
	}
	p.cb(requMsg, respMsg, event)
	return nil
}

func (p *parser) onDescribeGroupsTransaction(
	txid []byte,
	requMsg *requestMessage,
	respMsg *responseMessage,
) error {
	_, ok := parse.DescribeGroupsRequest(requMsg.payload)
	if !ok {
		return errParseFail
	}
	resp, ok := parse.DescribeGroupsResponse(respMsg.payload)
	if !ok {
		return errParseFail
	}

	for group, meta := range resp.Groups {
		event := common.MapStr{
			"group":          group,
			"status":         getStatus(meta.Error),
			"transaction_id": txid,
			"describe_group": common.MapStr{
				"response": common.MapStr{
					"error":         getError(meta.Error),
					"state":         meta.State,
					"protocol":      meta.Protocol,
					"protocol_type": meta.ProtocolType,
				},
			},
		}
		p.cb(requMsg, respMsg, event)
	}

	return nil
}

func (p *parser) onListGroupsTransaction(
	txid []byte,
	requMsg *requestMessage,
	respMsg *responseMessage,
) error {
	_, ok := parse.ListGroupRequest(requMsg.payload)
	if !ok {
		return errParseFail
	}
	resp, ok := parse.ListGroupResponse(respMsg.payload)
	if !ok {
		return errParseFail
	}

	groupInfo := make([]common.MapStr, 0, len(resp.Groups))
	for group, protocol := range resp.Groups {
		groupInfo = append(groupInfo, common.MapStr{
			"group":    group,
			"protocol": protocol,
		})
	}

	event := common.MapStr{
		"status":         getStatus(resp.Error),
		"transaction_id": txid,
		"list_groups": common.MapStr{
			"groups": groupInfo,
		},
	}
	p.cb(requMsg, respMsg, event)
	return nil
}

func (p *parser) onTransactionNoDetails(
	txid []byte,
	requMsg *requestMessage,
	respMsg *responseMessage,
) error {
	event := common.MapStr{
		"status":         "Unknown",
		"transaction_id": txid,
	}
	p.cb(requMsg, respMsg, event)
	return nil
}

func getStatus(err kafka.ErrorCode) string {
	if err == 0 {
		return common.OK_STATUS
	}
	return common.ERROR_STATUS
}

func getError(err kafka.ErrorCode) common.MapStr {
	return common.MapStr{
		"code": err,
	}
}

func makeTXID() []byte {
	return []byte(uuid.NewRandom())
}
