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

package amqp

import (
	"encoding/binary"
	"strconv"
	"strings"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
)

func connectionStartMethod(m *amqpMessage, args []byte) (bool, bool) {
	major := args[0]
	minor := args[1]
	properties := make(common.MapStr)
	next, err, exists := getTable(properties, args, 2)
	if err {
		// failed to get de peer-properties, size may be wrong, let's quit
		logp.Warn("Failed to parse server properties in connection.start method")
		return false, false
	}
	mechanisms, next, err := getShortString(args, next+4, binary.BigEndian.Uint32(args[next:next+4]))
	if err {
		logp.Warn("Failed to get connection mechanisms")
		return false, false
	}
	locales, _, err := getShortString(args, next+4, binary.BigEndian.Uint32(args[next:next+4]))
	if err {
		logp.Warn("Failed to get connection locales")
		return false, false
	}
	m.method = "connection.start"
	m.isRequest = true
	m.fields = common.MapStr{
		"version-major": major,
		"version-minor": minor,
		"mechanisms":    mechanisms,
		"locales":       locales,
	}
	// if there is a server properties table, add it
	if exists {
		m.fields["server-properties"] = properties
	}
	return true, true
}

func connectionStartOkMethod(m *amqpMessage, args []byte) (bool, bool) {
	properties := make(common.MapStr)
	next, err, exists := getTable(properties, args, 0)
	if err {
		// failed to get de peer-properties, size may be wrong, let's quit
		logp.Warn("Failed to parse server properties in connection.start method")
		return false, false
	}
	mechanism, next, err := getShortString(args, next+1, uint32(args[next]))
	if err {
		logp.Warn("Failed to get connection mechanism from client")
		return false, false
	}
	_, next, err = getShortString(args, next+4, binary.BigEndian.Uint32(args[next:next+4]))
	if err {
		logp.Warn("Failed to get connection response from client")
		return false, false
	}
	locale, _, err := getShortString(args, next+1, uint32(args[next]))
	if err {
		logp.Warn("Failed to get connection locale from client")
		return false, false
	}
	m.isRequest = false
	m.fields = common.MapStr{
		"mechanism": mechanism,
		"locale":    locale,
	}
	// if there is a client properties table, add it
	if exists {
		m.fields["client-properties"] = properties
	}
	return true, true
}

func connectionTuneMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.isRequest = true
	m.method = "connection.tune"
	// parameters are not parsed here, they are further negotiated by the server
	// in the connection.tune-ok method
	return true, true
}

func connectionTuneOkMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.fields = common.MapStr{
		"channel-max": binary.BigEndian.Uint16(args[0:2]),
		"frame-max":   binary.BigEndian.Uint32(args[2:6]),
		"heartbeat":   binary.BigEndian.Uint16(args[6:8]),
	}
	return true, true
}

func connectionOpenMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.isRequest = true
	m.method = "connection.open"
	host, _, err := getShortString(args, 1, uint32(args[0]))
	if err {
		logp.Warn("Failed to get virtual host from client")
		return false, false
	}
	m.fields = common.MapStr{"virtual-host": host}
	return true, true
}

func connectionCloseMethod(m *amqpMessage, args []byte) (bool, bool) {
	err := getCloseInfo(args, m)
	if err {
		return false, false
	}
	m.method = "connection.close"
	m.isRequest = true
	return true, true
}

func channelOpenMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.method = "channel.open"
	m.isRequest = true
	return true, true
}

func channelFlowMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.method = "channel.flow"
	m.isRequest = true
	return true, true
}

func channelFlowOkMethod(m *amqpMessage, args []byte) (bool, bool) {
	params := getBitParams(args[0])
	m.fields = common.MapStr{"active": params[0]}
	return true, true
}

func channelCloseMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.method = "channel.close"
	m.isRequest = true
	err := getCloseInfo(args, m)
	if err {
		return false, false
	}
	return true, true
}

// function to fetch fields from channel close and connection close
func getCloseInfo(args []byte, m *amqpMessage) bool {
	code := binary.BigEndian.Uint16(args[0:2])
	m.isRequest = true
	replyText, nextOffset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Failed to get error reply text")
		return true
	}
	m.fields = common.MapStr{
		"reply-code": code,
		"reply-text": replyText,
		"class-id":   binary.BigEndian.Uint16(args[nextOffset : nextOffset+2]),
		"method-id":  binary.BigEndian.Uint16(args[nextOffset+2 : nextOffset+4]),
	}
	return false
}

func queueDeclareMethod(m *amqpMessage, args []byte) (bool, bool) {
	name, offset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of queue in queue declaration")
		return false, false
	}
	m.isRequest = true
	m.method = "queue.declare"
	params := getBitParams(args[offset])
	m.request = name
	m.fields = common.MapStr{
		"queue":       name,
		"passive":     params[0],
		"durable":     params[1],
		"exclusive":   params[2],
		"auto-delete": params[3],
		"no-wait":     params[4],
	}
	if args[offset+1] != frameEndOctet && m.parseArguments {
		arguments := make(common.MapStr)
		_, err, exists := getTable(arguments, args, offset+1)
		if !err && exists {
			m.fields["arguments"] = arguments
		} else if err {
			m.notes = append(m.notes, "Failed to parse additional arguments")
		}
	}
	return true, true
}

func queueDeclareOkMethod(m *amqpMessage, args []byte) (bool, bool) {
	name, nextOffset, err := getShortString(args, 1, uint32(args[0]))
	if err {
		logp.Warn("Error getting name of queue in queue confirmation")
		return false, false
	}
	m.method = "queue.declare-ok"
	m.fields = common.MapStr{
		"queue":          name,
		"consumer-count": binary.BigEndian.Uint32(args[nextOffset+4:]),
		"message-count":  binary.BigEndian.Uint32(args[nextOffset : nextOffset+4]),
	}
	return true, true
}

func queueBindMethod(m *amqpMessage, args []byte) (bool, bool) {
	queue, offset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of queue in queue bind")
		return false, false
	}
	m.isRequest = true
	exchange, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Error getting name of queue in queue bind")
		return false, false
	}
	routingKey, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Error getting name of queue in queue bind")
		return false, false
	}
	params := getBitParams(args[offset])
	m.method = "queue.bind"
	m.request = strings.Join([]string{queue, exchange}, " ")
	m.fields = common.MapStr{
		"queue":       queue,
		"routing-key": routingKey,
		"no-wait":     params[0],
	}
	if len(exchange) > 0 {
		m.fields["exchange"] = exchange
	}
	if args[offset+1] != frameEndOctet && m.parseArguments {
		arguments := make(common.MapStr)
		_, err, exists := getTable(arguments, args, offset+1)
		if !err && exists {
			m.fields["arguments"] = arguments
		} else if err {
			m.notes = append(m.notes, "Failed to parse additional arguments")
		}
	}
	return true, true
}

func queueUnbindMethod(m *amqpMessage, args []byte) (bool, bool) {
	queue, offset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of queue in queue unbind")
		return false, false
	}
	exchange, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Error getting name of queue in queue unbind")
		return false, false
	}
	routingKey, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Error getting name of queue in queue unbind")
		return false, false
	}
	m.isRequest = true
	m.method = "queue.unbind"
	m.request = strings.Join([]string{queue, exchange}, " ")
	m.fields = common.MapStr{
		"queue":       queue,
		"routing-key": routingKey,
	}
	if len(exchange) > 0 {
		m.fields["exchange"] = exchange
	}
	if args[offset+1] != frameEndOctet && m.parseArguments {
		arguments := make(common.MapStr)
		_, err, exists := getTable(arguments, args, offset+1)
		if !err && exists {
			m.fields["arguments"] = arguments
		} else if err {
			m.notes = append(m.notes, "Failed to parse additional arguments")
		}
	}
	return true, true
}

func queuePurgeMethod(m *amqpMessage, args []byte) (bool, bool) {
	queue, nextOffset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of queue in queue purge")
		return false, false
	}
	m.isRequest = true
	params := getBitParams(args[nextOffset])
	m.method = "queue.purge"
	m.request = queue
	m.fields = common.MapStr{
		"queue":   queue,
		"no-wait": params[0],
	}
	return true, true
}

func queuePurgeOkMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.method = "queue.purge-ok"
	m.fields = common.MapStr{
		"message-count": binary.BigEndian.Uint32(args[0:4]),
	}
	return true, true
}

func queueDeleteMethod(m *amqpMessage, args []byte) (bool, bool) {
	queue, nextOffset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of queue in queue delete")
		return false, false
	}
	m.isRequest = true
	params := getBitParams(args[nextOffset])
	m.method = "queue.delete"
	m.request = queue
	m.fields = common.MapStr{
		"queue":     queue,
		"if-unused": params[0],
		"if-empty":  params[1],
		"no-wait":   params[2],
	}
	return true, true
}

func queueDeleteOkMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.method = "queue.delete-ok"
	m.fields = common.MapStr{
		"message-count": binary.BigEndian.Uint32(args[0:4]),
	}
	return true, true
}

func exchangeDeclareMethod(m *amqpMessage, args []byte) (bool, bool) {
	exchange, offset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of exchange in exchange declare")
		return false, false
	}
	exchangeType, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Error getting name of routing key in exchange declare")
		return false, false
	}
	params := getBitParams(args[offset])
	m.method = "exchange.declare"
	m.isRequest = true
	m.request = exchange
	if exchangeType == "" {
		exchangeType = "direct"
	}
	m.fields = common.MapStr{
		"exchange":      exchange,
		"exchange-type": exchangeType,
		"passive":       params[0],
		"durable":       params[1],
		"no-wait":       params[4],
	}
	if args[offset+1] != frameEndOctet && m.parseArguments {
		arguments := make(common.MapStr)
		_, err, exists := getTable(arguments, args, offset+1)
		if !err && exists {
			m.fields["arguments"] = arguments
		} else if err {
			m.notes = append(m.notes, "Failed to parse additional arguments")
		}
	}
	return true, true
}

func exchangeDeleteMethod(m *amqpMessage, args []byte) (bool, bool) {
	exchange, nextOffset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of exchange in exchange delete")
		return false, false
	}
	m.method = "exchange.delete"
	m.isRequest = true
	params := getBitParams(args[nextOffset])
	m.request = exchange
	m.fields = common.MapStr{
		"exchange":  exchange,
		"if-unused": params[0],
		"no-wait":   params[1],
	}
	return true, true
}

// this is a method exclusive to RabbitMQ
func exchangeBindMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.method = "exchange.bind"
	err := exchangeBindUnbindInfo(m, args)
	if err {
		return false, false
	}
	return true, true
}

// this is a method exclusive to RabbitMQ
func exchangeUnbindMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.method = "exchange.unbind"
	err := exchangeBindUnbindInfo(m, args)
	if err {
		return false, false
	}
	return true, true
}

func exchangeBindUnbindInfo(m *amqpMessage, args []byte) bool {
	destination, offset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of destination in exchange bind/unbind")
		return true
	}
	source, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Error getting name of source in exchange bind/unbind")
		return true
	}
	routingKey, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Error getting name of routing-key in exchange bind/unbind")
		return true
	}
	m.isRequest = true
	params := getBitParams(args[offset])
	m.request = strings.Join([]string{source, destination}, " ")
	m.fields = common.MapStr{
		"destination": destination,
		"source":      source,
		"routing-key": routingKey,
		"no-wait":     params[0],
	}
	if args[offset+1] != frameEndOctet && m.parseArguments {
		arguments := make(common.MapStr)
		_, err, exists := getTable(arguments, args, offset+1)
		if !err && exists {
			m.fields["arguments"] = arguments
		} else if err {
			m.notes = append(m.notes, "Failed to parse additional arguments")
		}
	}
	return false
}

func basicQosMethod(m *amqpMessage, args []byte) (bool, bool) {
	prefetchSize := binary.BigEndian.Uint32(args[0:4])
	prefetchCount := binary.BigEndian.Uint16(args[4:6])
	params := getBitParams(args[6])
	m.isRequest = true
	m.method = "basic.qos"
	m.fields = common.MapStr{
		"prefetch-size":  prefetchSize,
		"prefetch-count": prefetchCount,
		"global":         params[0],
	}
	return true, true
}

func basicConsumeMethod(m *amqpMessage, args []byte) (bool, bool) {
	queue, offset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of queue in basic consume")
		return false, false
	}
	consumerTag, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Error getting name of consumer tag in basic consume")
		return false, false
	}
	params := getBitParams(args[offset])
	m.method = "basic.consume"
	m.isRequest = true
	m.request = queue
	m.fields = common.MapStr{
		"queue":        queue,
		"consumer-tag": consumerTag,
		"no-local":     params[0],
		"no-ack":       params[1],
		"exclusive":    params[2],
		"no-wait":      params[3],
	}
	if args[offset+1] != frameEndOctet && m.parseArguments {
		arguments := make(common.MapStr)
		_, err, exists := getTable(arguments, args, offset+1)
		if !err && exists {
			m.fields["arguments"] = arguments
		} else if err {
			m.notes = append(m.notes, "Failed to parse additional arguments")
		}
	}
	return true, true
}

func basicConsumeOkMethod(m *amqpMessage, args []byte) (bool, bool) {
	consumerTag, _, err := getShortString(args, 1, uint32(args[0]))
	if err {
		logp.Warn("Error getting name of queue in basic consume")
		return false, false
	}
	m.method = "basic.consume-ok"
	m.fields = common.MapStr{
		"consumer-tag": consumerTag,
	}
	return true, true
}

func basicCancelMethod(m *amqpMessage, args []byte) (bool, bool) {
	consumerTag, offset, err := getShortString(args, 1, uint32(args[0]))
	if err {
		logp.Warn("Error getting consumer tag in basic cancel")
		return false, false
	}
	m.method = "basic.cancel"
	m.isRequest = true
	m.request = consumerTag
	params := getBitParams(args[offset])
	m.fields = common.MapStr{
		"consumer-tag": consumerTag,
		"no-wait":      params[0],
	}
	return true, true
}

func basicCancelOkMethod(m *amqpMessage, args []byte) (bool, bool) {
	consumerTag, _, err := getShortString(args, 1, uint32(args[0]))
	if err {
		logp.Warn("Error getting consumer tag in basic cancel ok")
		return false, false
	}
	m.method = "basic.cancel-ok"
	m.fields = common.MapStr{
		"consumer-tag": consumerTag,
	}
	return true, true
}

func basicPublishMethod(m *amqpMessage, args []byte) (bool, bool) {
	exchange, nextOffset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting exchange in basic publish")
		return false, false
	}
	routingKey, nextOffset, err := getShortString(args, nextOffset+1, uint32(args[nextOffset]))
	if err {
		logp.Warn("Error getting routing key in basic publish")
		return false, false
	}
	params := getBitParams(args[nextOffset])
	m.method = "basic.publish"
	m.fields = common.MapStr{
		"routing-key": routingKey,
		"mandatory":   params[0],
		"immediate":   params[1],
	}
	// is exchange not default exchange ?
	if len(exchange) > 0 {
		m.fields["exchange"] = exchange
	}
	return true, false
}

func basicReturnMethod(m *amqpMessage, args []byte) (bool, bool) {
	code := binary.BigEndian.Uint16(args[0:2])
	if code < 300 {
		// not an error or exception ? not interesting
		return true, false
	}
	replyText, nextOffset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of reply text in basic return")
		return false, false
	}
	exchange, nextOffset, err := getShortString(args, nextOffset+1, uint32(args[nextOffset]))
	if err {
		logp.Warn("Error getting name of exchange in basic return")
		return false, false
	}
	routingKey, _, err := getShortString(args, nextOffset+1, uint32(args[nextOffset]))
	if err {
		logp.Warn("Error getting name of routing key in basic return")
		return false, false
	}
	m.method = "basic.return"
	m.fields = common.MapStr{
		"exchange":    exchange,
		"routing-key": routingKey,
		"reply-code":  code,
		"reply-text":  replyText,
	}
	return true, false
}

func basicDeliverMethod(m *amqpMessage, args []byte) (bool, bool) {
	consumerTag, offset, err := getShortString(args, 1, uint32(args[0]))
	if err {
		logp.Warn("Failed to get consumer tag in basic deliver")
		return false, false
	}
	deliveryTag := binary.BigEndian.Uint64(args[offset : offset+8])
	params := getBitParams(args[offset+8])
	offset = offset + 9
	exchange, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Failed to get exchange in basic deliver")
		return false, false
	}
	routingKey, _, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Failed to get routing key in basic deliver")
		return false, false
	}
	m.method = "basic.deliver"
	m.fields = common.MapStr{
		"consumer-tag": consumerTag,
		"delivery-tag": deliveryTag,
		"redelivered":  params[0],
		"routing-key":  routingKey,
	}
	// is exchange not default exchange ?
	if len(exchange) > 0 {
		m.fields["exchange"] = exchange
	}
	return true, false
}

func basicGetMethod(m *amqpMessage, args []byte) (bool, bool) {
	queue, offset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Failed to get queue in basic get method")
		return false, false
	}
	m.method = "basic.get"
	params := getBitParams(args[offset])
	m.isRequest = true
	m.request = queue
	m.fields = common.MapStr{
		"queue":  queue,
		"no-ack": params[0],
	}
	return true, true
}

func basicGetOkMethod(m *amqpMessage, args []byte) (bool, bool) {
	params := getBitParams(args[8])
	exchange, offset, err := getShortString(args, 10, uint32(args[9]))
	if err {
		logp.Warn("Failed to get queue in basic get-ok")
		return false, false
	}
	routingKey, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Failed to get routing key in basic get-ok")
		return false, false
	}
	m.method = "basic.get-ok"
	m.fields = common.MapStr{
		"delivery-tag":  binary.BigEndian.Uint64(args[0:8]),
		"redelivered":   params[0],
		"routing-key":   routingKey,
		"message-count": binary.BigEndian.Uint32(args[offset : offset+4]),
	}
	if len(exchange) > 0 {
		m.fields["exchange"] = exchange
	}
	return true, false
}

func basicGetEmptyMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.method = "basic.get-empty"
	return true, true
}

func basicAckMethod(m *amqpMessage, args []byte) (bool, bool) {
	params := getBitParams(args[8])
	m.method = "basic.ack"
	m.isRequest = true
	m.fields = common.MapStr{
		"delivery-tag": binary.BigEndian.Uint64(args[0:8]),
		"multiple":     params[0],
	}
	return true, true
}

// this is a rabbitMQ specific method
func basicNackMethod(m *amqpMessage, args []byte) (bool, bool) {
	params := getBitParams(args[8])
	m.method = "basic.nack"
	m.isRequest = true
	m.fields = common.MapStr{
		"delivery-tag": binary.BigEndian.Uint64(args[0:8]),
		"multiple":     params[0],
		"requeue":      params[1],
	}
	return true, true
}

func basicRejectMethod(m *amqpMessage, args []byte) (bool, bool) {
	params := getBitParams(args[8])
	tag := binary.BigEndian.Uint64(args[0:8])
	m.isRequest = true
	m.method = "basic.reject"
	m.fields = common.MapStr{
		"delivery-tag": tag,
		"multiple":     params[0],
	}
	m.request = strconv.FormatUint(tag, 10)
	return true, true
}

func basicRecoverMethod(m *amqpMessage, args []byte) (bool, bool) {
	params := getBitParams(args[0])
	m.isRequest = true
	m.method = "basic.recover"
	m.fields = common.MapStr{
		"requeue": params[0],
	}
	return true, true
}

func txSelectMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.isRequest = true
	m.method = "tx.select"
	return true, true
}

func txCommitMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.isRequest = true
	m.method = "tx.commit"
	return true, true
}

func txRollbackMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.isRequest = true
	m.method = "tx.rollback"
	return true, true
}

// simple function used when server/client responds to a sync method with no new info
func okMethod(m *amqpMessage, args []byte) (bool, bool) {
	return true, true
}

// function to get a short string. It sends back an error if slice is too short
// for declared length. if length == 0, the function sends back an empty string and
// advances the offset. Otherwise, it returns the string and the new offset
func getShortString(data []byte, start uint32, length uint32) (short string, nextOffset uint32, err bool) {
	if length == 0 {
		return "", start, false
	}
	if uint32(len(data)) < start || uint32(len(data[start:])) < length {
		return "", 0, true
	}
	return string(data[start : start+length]), start + length, false
}

// function to extract bit information in various AMQP methods
func getBitParams(bits byte) (ret [5]bool) {
	if bits&16 == 16 {
		ret[4] = true
	}
	if bits&8 == 8 {
		ret[3] = true
	}
	if bits&4 == 4 {
		ret[2] = true
	}
	if bits&2 == 2 {
		ret[1] = true
	}
	if bits&1 == 1 {
		ret[0] = true
	}
	return ret
}
